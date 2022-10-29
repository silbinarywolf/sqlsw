package dbreflect

import (
	"database/sql"
	"reflect"
)

type Value struct {
	value reflect.Value
}

func (v *Value) UnderlyingValue() reflect.Value {
	return v.value
}

func (v *Value) Interface() interface{} {
	return v.value.Interface()
}

// Indirect returns the value that v points to.
// If v is a nil pointer, Indirect returns a zero Value.
// If v is not a pointer, Indirect returns v.
func Indirect(value Value) Value {
	v := Value{}
	v.value = reflect.Indirect(value.value)
	return v
}

// ValueOf returns a new Value initialized to the concrete value
// stored in the interface i. ValueOf(nil) returns the zero Value.
//
// note(jae): 2022-10-15
// wrapping this so we can potentially quickly replace with a faster reflect
// library
func ValueOf(value interface{}) Value {
	v := Value{}
	v.value = reflect.ValueOf(value)
	return v
}

type Type struct {
	typ reflect.Type
}

func TypeOf(value interface{}) Type {
	v := Type{}
	v.typ = reflect.TypeOf(value)
	return v
}

// PtrTo returns the pointer type with element t.
// For example, if t represents type Foo, PtrTo(t) represents *Foo.
//
// PtrTo is the old spelling of PointerTo.
// The two functions behave identically.
func PtrTo(typ Type) Type {
	v := Type{}
	v.typ = reflect.PtrTo(typ.typ)
	return v
}

// New returns a Value representing a pointer to a new zero value
// for the specified type. That is, the returned Value's Type is PointerTo(typ).
func New(typ Type) Value {
	v := Value{}
	v.value = reflect.New(typ.typ)
	return v
}

// Implements reports whether the type implements the interface type u.
func (typ Type) Implements(otherTyp Type) bool {
	return typ.typ.Implements(otherTyp.typ)
}

func (typ Type) Kind() reflect.Kind {
	return typ.typ.Kind()
}

func (typ Type) Elem() Type {
	v := Type{}
	v.typ = typ.typ.Elem()
	return v
}

// NumField returns a struct type's field count.
// It panics if the type's Kind is not Struct.
func (typ Type) NumField() int {
	return typ.typ.NumField()
}

func (typ Type) Field(i int) structField {
	v := structField{}
	v.field = typ.typ.Field(i)
	return v
}

type structField struct {
	field reflect.StructField
}

// Anonymous is an embedded field
func (structField structField) Anonymous() bool {
	return structField.field.Anonymous
}

func (structField structField) Tag() reflect.StructTag {
	return structField.field.Tag
}

func (structField structField) Type() Type {
	v := Type{}
	v.typ = structField.field.Type
	return v
}

func (structField structField) PkgPath() string {
	// PkgPath is the package path that qualifies a lower case (unexported)
	// field name. It is empty for upper case (exported) field names.
	// See https://golang.org/ref/spec#Uniqueness_of_identifiers
	return structField.field.PkgPath
}

var _scannerInterface = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

func (typ Type) IsScannable() bool {
	if reflect.PtrTo(typ.typ).Implements(_scannerInterface) {
		return true
	}
	// If not a struct, then it's likely a supported primitive that
	// can be scanned by database/sql
	if typ.Kind() != reflect.Struct {
		return true
	}
	// If struct and has no exported fields, it's scannable.
	//
	// note(jae): 2022-10-29
	// This test mostly exists to ensure time.Time works, at least
	// at time of writing.
	{
		hasExportedField := false
		numFields := typ.typ.NumField()
		for i := 0; i < numFields; i++ {
			hasExportedField = hasExportedField || typ.typ.Field(i).IsExported()
		}
		if !hasExportedField {
			return true
		}
	}
	return false
}
