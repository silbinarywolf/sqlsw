// dbreflect is used to access struct fields for database purposes
package dbreflect

import (
	"fmt"
	"reflect"
	"sync"
)

var defaultModule = ReflectModule{}

type ReflectModule struct {
	cachedStructs sync.Map
}

type ReflectProcessor struct {
	fields []StructField
	errors []error
}

type StructField struct {
	tagName      string
	reflectValue reflect.Value
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

func (field *StructField) Interface() interface{} {
	return field.reflectValue.Interface()
}

type reflectProcessErrorList struct {
	errors []error
}

func (err *reflectProcessErrorList) Error() string {
	// todo(jae): 2022-10-15
	// print each on a line
	return fmt.Sprintf("%+v", err.errors)
}

func GetStruct(valueEl reflect.Value) (*Struct, error) {
	return defaultModule.GetStruct(valueEl)
}

func (m *ReflectModule) GetStruct(valueEl reflect.Value) (*Struct, error) {
	key := valueEl.Type()
	unassertedStructInfo, ok := m.cachedStructs.Load(key)
	if ok {
		return unassertedStructInfo.(*Struct), nil
	}
	structInfo, err := getStruct(valueEl)
	if err != nil {
		return nil, err
	}
	m.cachedStructs.Store(key, &structInfo)
	return &structInfo, nil
}

func getStruct(valueEl reflect.Value) (Struct, error) {
	p := ReflectProcessor{}
	p.processFields(valueEl)
	if len(p.errors) > 0 {
		return Struct{}, &reflectProcessErrorList{errors: p.errors}
	}
	struc := Struct{}
	struc.fields = p.fields
	return struc, nil
}

func (p *ReflectProcessor) processFields(valueEl reflect.Value) {
	typeEl := valueEl.Type()
	structFieldLen := typeEl.NumField()
	for i := 0; i < structFieldLen; i++ {
		structField := valueEl.Field(i)
		// note(jae): 2022-10-15
		// getting reflect.StructField causes 1-alloc
		structFieldType := typeEl.Field(i)
		if structFieldType.Anonymous {
			fieldType := structFieldType.Type
			fieldTypeKind := fieldType.Kind()
			if fieldTypeKind == reflect.Struct {
				p.processFields(structField)
				continue
			}
			if fieldTypeKind == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
				p.processFields(structField.Elem())
				continue
			}
		}
		// note(jae): 2022-10-15
		// This check must happen *after* the "Anonymous" check so
		// that embedding unexported structs within a struct still works
		// if !structField.CanSet() {
		// 	continue
		// }
		fullTagInfo, ok := structFieldType.Tag.Lookup("db")
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
		field.reflectValue = structField
		p.fields = append(p.fields, field)
	}
}
