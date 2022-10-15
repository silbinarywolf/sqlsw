// dbreflect is used to access struct fields for database purposes
package dbreflect

import (
	"fmt"
	"reflect"
	"sync"
)

type ReflectModule struct {
	cachedStructs sync.Map
}

type ReflectProcessor struct {
	fields []StructField
	// indexes represents the current field index depth
	indexes []int
	errors  []error
}

type StructField struct {
	tagName string
	// indexes represents the current field index depth
	indexes []int

	indexesUnderlying [4]int
}

type Struct struct {
	fields []StructField
}

func (struc *Struct) GetFieldByTagName(dbTagName string) (*StructField, bool) {
	for i := range struc.fields {
		field := &struc.fields[i]
		if field.tagName == dbTagName {
			return field, true
		}
	}
	return nil, false
}

// Interface returns the struct field value using the provided struct
func (field *StructField) Interface(structAsReflectValue Value) interface{} {
	v := structAsReflectValue.value
	for _, i := range field.indexes {
		v = reflect.Indirect(v).Field(i)
	}
	return v.Interface()
}

// Addr returns the address of the struct field
func (field *StructField) Addr(structAsReflectValue Value) interface{} {
	v := structAsReflectValue.value
	for _, i := range field.indexes {
		v = reflect.Indirect(v).Field(i)
	}
	return v.Addr().Interface()
}

// SetValue will set the value on the struct using the value
/* func (field *StructField) SetValue(structAsReflectValue Value, value interface{}) {
	v := structAsReflectValue.value
	for _, i := range field.indexes {
		v = reflect.Indirect(v).Field(i)
	}
	v.Set(reflect.ValueOf(value))
} */

type reflectProcessErrorList struct {
	errors []error
}

func (err *reflectProcessErrorList) Error() string {
	// todo(jae): 2022-10-15
	// print each on a line
	return fmt.Sprintf("%+v", err.errors)
}

func (m *ReflectModule) GetStruct(typeEl Type) (*Struct, error) {
	structInfo, err := getStruct(typeEl)
	if err != nil {
		return nil, err
	}
	return &structInfo, nil
	/* key := typeEl
	unassertedStructInfo, ok := m.cachedStructs.Load(key)
	if ok {
		return unassertedStructInfo.(*Struct), nil
	}
	structInfo, err := getStruct(typeEl)
	if err != nil {
		return nil, err
	}
	m.cachedStructs.Store(key, &structInfo)
	return &structInfo, nil */
}

func getStruct(typeEl Type) (Struct, error) {
	var indexesUnderlying [8]int
	p := ReflectProcessor{}
	p.indexes = indexesUnderlying[:0]
	p.processFields(typeEl)
	if len(p.errors) > 0 {
		return Struct{}, &reflectProcessErrorList{errors: p.errors}
	}
	struc := Struct{}
	struc.fields = p.fields
	return struc, nil
}

func (p *ReflectProcessor) processFields(typeEl Type) {
	structFieldLen := typeEl.NumField()
	for i := 0; i < structFieldLen; i++ {
		// note(jae): 2022-10-15
		// getting reflect.StructField causes 1-alloc
		structFieldType := typeEl.Field(i)
		if structFieldType.Anonymous() {
			fieldType := structFieldType.Type()
			fieldTypeKind := fieldType.Kind()
			if fieldTypeKind == reflect.Struct {
				p.indexes = append(p.indexes, i)
				p.processFields(fieldType)
				p.indexes = p.indexes[:len(p.indexes)-1]
				continue
			}
			if fieldTypeKind == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
				p.indexes = append(p.indexes, i)
				p.processFields(fieldType.Elem())
				p.indexes = p.indexes[:len(p.indexes)-1]
				continue
			}
		}
		// note(jae): 2022-10-15
		// This check must happen *after* the "Anonymous" check so
		// that embedding unexported structs within a struct still works
		// if !structField.CanSet() {
		// 	continue
		// }
		fullTagInfo, ok := structFieldType.Tag().Lookup("db")
		if !ok {
			// skip if there is no tag on field
			continue
		}
		if fullTagInfo == "-" {
			// skip if tag value is "-"
			// ie. `db:"-"`
			continue
		}
		// Get tag name
		tagName := fullTagInfo
	TagLoop:
		for pos := 0; pos < len(fullTagInfo); {
			c := fullTagInfo[pos]
			switch c {
			case ',':
				tagName = fullTagInfo[0:pos]
			case ':':
				// If unexpected tag value, ignore
				//
				// note(jae): 2022-10-15
				// This behaviour is retained from sqlx
				tagName = ""
				break TagLoop
			}
			pos += 1
		}
		if tagName == "" {
			continue
		}
		field := StructField{}
		field.tagName = tagName
		field.indexes = field.indexesUnderlying[:0]
		field.indexes = append(field.indexes, p.indexes...)
		field.indexes = append(field.indexes, i)
		p.fields = append(p.fields, field)
	}
}
