package dbreflect

import (
	"reflect"
)

type Value struct {
	value reflect.Value
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

func (typ Type) Kind() reflect.Kind {
	return typ.typ.Kind()
}

func (typ Type) Elem() Type {
	v := Type{}
	v.typ = typ.typ.Elem()
	return v
}

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
