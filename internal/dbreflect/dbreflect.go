// dbreflect is used to access struct fields for database purposes
package dbreflect

import (
	"bytes"
	"reflect"
	"strings"
	"sync"
)

type ReflectModule struct {
	cachedStructs sync.Map
	options       Options
}

func NewReflectModule(options Options) *ReflectModule {
	m := &ReflectModule{}
	m.options = options
	return m
}

type Options struct {
	LowercaseFieldNameWithNoTag bool
}

type ReflectProcessor struct {
	fields []StructField
	// indexes represents the current field index depth
	indexes []int
	errors  []error
	// Options are the config options
	Options
}

type StructField struct {
	tagName string
	// indexes represents the current field index depth
	// ie. [0] = first field of struct, [0][1] = first field of struct, second field of sub-struct
	indexes []int

	indexesUnderlying [4]int
}

type Struct struct {
	fields []StructField
}

func (struc *Struct) GetFieldByName(fieldName string) (*StructField, bool) {
	for i := range struc.fields {
		field := &struc.fields[i]
		if field.tagName == fieldName {
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

type reflectProcessErrorList struct {
	errors []error
}

func (err *reflectProcessErrorList) Error() string {
	if len(err.errors) == 0 {
		return "missing error information"
	}
	if len(err.errors) == 1 {
		return err.errors[0].Error()
	}
	var buf bytes.Buffer
	buf.WriteString("Multiple reflection errors:\n")
	for _, subErr := range err.errors {
		buf.WriteString("- ")
		buf.WriteString(subErr.Error())
		buf.WriteRune('\n')
	}
	return buf.String()
}

func (m *ReflectModule) GetStruct(typeEl Type) (*Struct, error) {
	// uncached
	/* structInfo, err := getStruct(typeEl)
	if err != nil {
		return nil, err
	}
	return &structInfo, nil */
	// cached
	key := typeEl
	unassertedStructInfo, ok := m.cachedStructs.Load(key)
	if ok {
		return unassertedStructInfo.(*Struct), nil
	}
	structInfo, err := getStruct(typeEl, m.options)
	if err != nil {
		return nil, err
	}
	m.cachedStructs.Store(key, &structInfo)
	return &structInfo, nil
}

func getStruct(typeEl Type, options Options) (Struct, error) {
	var indexesUnderlying [8]int
	p := ReflectProcessor{}
	p.Options = options
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
		// note(jae): 2022-10-23
		// I forgot why we don't do this check anymore but if we uncomment
		// this, something breaks.
		// if !structField.CanSet() {
		// 	continue
		// }

		// todo(jae): 2022-10-16
		// sqlx does not skip if there is no tag, we need to add compatibility
		// for that here in the compat layer.
		fullTagInfo, ok := structFieldType.Tag().Lookup("db")
		var dbFieldName string
		if !ok {
			if !p.LowercaseFieldNameWithNoTag {
				// Skip field if there is no tag
				continue
			}
			// note(jae): 2022-10-23
			// SQLX Default mapper backwards compatibility
			dbFieldName = strings.ToLower(structFieldType.field.Name)
		} else {
			if fullTagInfo == "-" {
				// skip if tag value is "-"
				// ie. `db:"-"`
				continue
			}
			// Get tag name
			dbFieldName = fullTagInfo
		TagLoop:
			for pos := 0; pos < len(fullTagInfo); {
				c := fullTagInfo[pos]
				switch c {
				case ',':
					dbFieldName = fullTagInfo[0:pos]
				case ':':
					// If unexpected tag value, ignore
					//
					// note(jae): 2022-10-15
					// This behaviour is retained from sqlx
					dbFieldName = ""
					break TagLoop
				}
				pos += 1
			}
		}
		if dbFieldName == "" {
			continue
		}
		field := StructField{}
		field.tagName = dbFieldName
		field.indexes = field.indexesUnderlying[:0]
		field.indexes = append(field.indexes, p.indexes...)
		field.indexes = append(field.indexes, i)
		p.fields = append(p.fields, field)
	}
}

// ResetCache is used by benchmarking tests
func (m *ReflectModule) ResetCache() {
	m.cachedStructs = sync.Map{}
}
