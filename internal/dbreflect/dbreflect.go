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
	// dbFieldNames holds depth of struct names
	// ie. struct { MyField OtherStruct `db:"otherStruct"` } = ["otherStruct"]
	dbFieldNames []string
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

// DebugFieldNames will return all field names on the struct
func (struc *Struct) DebugFieldNames() []string {
	r := make([]string, 0, len(struc.fields))
	for i := range struc.fields {
		field := &struc.fields[i]
		r = append(r, field.tagName)
	}
	return r
}

// Interface returns the struct field value using the provided struct
func (field *StructField) Interface(structAsReflectValue Value) interface{} {
	v := structAsReflectValue.value
	for _, i := range field.indexes {
		v = reflect.Indirect(v).Field(i)
	}
	return v.Interface()
}

// AddrWithNew returns the address of the struct field
//
// If it's nil pointer or map, it will allocate a new one within the struct
func (field *StructField) AddrWithNew(structAsReflectValue Value) interface{} {
	v := structAsReflectValue.value
	for _, i := range field.indexes {
		v = reflect.Indirect(v).Field(i)
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
		case reflect.Map:
			panic("todo(jae): Test this proper")
			/* if v.IsNil() {
				v.Set(reflect.MakeMap(v.Type().Elem()))
			} */
		}
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

		// Determine database field name by:
		// - Getting it from `db:""` tag
		// - Fallback to using the struct field name and auto-lowercasing (SQLX compatibility)
		fullTagInfo, ok := structFieldType.Tag().Lookup("db")
		var dbFieldName string
		hasTagName := false
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
			hasTagName = true
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

		// Process structs
		{
			if structFieldType.Anonymous() {
				fieldType := structFieldType.Type()
				fieldTypeKind := fieldType.Kind()
				if fieldTypeKind == reflect.Struct {
					// note(jae): 2022-10-29
					// Only append field name information to the depth for embedded
					// structs if there's an explicit tag on it.
					//
					// This handles SQLX backwards compatibility, see "TestJoinQuery" in sqlx_test.go
					if hasTagName {
						p.dbFieldNames = append(p.dbFieldNames, dbFieldName)
					}
					p.indexes = append(p.indexes, i)
					p.processFields(fieldType)
					if hasTagName {
						p.dbFieldNames = p.dbFieldNames[:len(p.dbFieldNames)-1]
					}
					p.indexes = p.indexes[:len(p.indexes)-1]

					// Skip to next field
					continue
				}
				if fieldTypeKind == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
					// note(jae): 2022-10-29
					// Only append field name information to the depth for embedded
					// structs if there's an explicit tag on it.
					//
					// This handles SQLX backwards compatibility, see "TestJoinQuery" in sqlx_test.go
					if hasTagName {
						p.dbFieldNames = append(p.dbFieldNames, dbFieldName)
					}
					p.indexes = append(p.indexes, i)
					p.processFields(fieldType.Elem())
					if hasTagName {
						p.dbFieldNames = p.dbFieldNames[:len(p.dbFieldNames)-1]
					}
					p.indexes = p.indexes[:len(p.indexes)-1]

					// Skip to next field
					continue
				}
			}
			fieldType := structFieldType.Type()
			fieldTypeKind := fieldType.Kind()
			if fieldTypeKind == reflect.Struct {
				// note(jae): 2022-10-29
				// Avoid sub-processing of structs that implement `Scan(any) error`.
				// Without this, structs like sql.NullInt64 won't work as expected.
				if isScannable := fieldType.IsScannable(); !isScannable {
					// Push to stack
					p.dbFieldNames = append(p.dbFieldNames, dbFieldName)
					p.indexes = append(p.indexes, i)

					// Process the structs fields
					p.processFields(fieldType)

					// Pop from stack
					p.dbFieldNames = p.dbFieldNames[:len(p.dbFieldNames)-1]
					p.indexes = p.indexes[:len(p.indexes)-1]

					// Skip to next field
					continue
				}
			}
			if fieldTypeKind == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
				// note(jae): 2022-10-29
				// Avoid sub-processing of structs that implement `Scan(any) error`.
				// Without this, structs like sql.NullInt64 won't work as expected.
				if isScannable := fieldType.Elem().IsScannable(); !isScannable {
					// Push to stack
					p.dbFieldNames = append(p.dbFieldNames, dbFieldName)
					p.indexes = append(p.indexes, i)

					// Process the structs fields
					p.processFields(fieldType.Elem())

					// Pop from stack
					p.dbFieldNames = p.dbFieldNames[:len(p.dbFieldNames)-1]
					p.indexes = p.indexes[:len(p.indexes)-1]

					// Skip to next field
					continue
				}
			}
		}
		field := StructField{}
		if len(p.dbFieldNames) == 0 {
			field.tagName = dbFieldName
		} else {
			// Handle case where non-embedded structs are used
			// ie. "myStruct.id" or "myStruct.deeperStruct.id"
			var buf bytes.Buffer
			for _, subFieldName := range p.dbFieldNames {
				buf.WriteString(subFieldName + ".")
			}
			field.tagName = buf.String() + dbFieldName
		}
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
