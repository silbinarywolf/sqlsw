package sqlsw

import (
	"reflect"
	"testing"
)

// Go 1.19
// - 1000000000	         0.2484 ns/op	       0 B/op	       0 allocs/op
func BenchmarkTypeAssertConvertToMapStringInterface(b *testing.B) {
	var v interface{}
	{
		v = map[string]interface{}{
			"ID":   1,
			"Name": "Title",
		}
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, ok := v.(map[string]interface{})
		if !ok {
			b.Fatal("Failed to convert")
		}
	}
}

// Go 1.19
// - 44238000	        26.20 ns/op	       0 B/op	       0 allocs/op
func BenchmarkConvertToMapStringInterface(b *testing.B) {
	var v interface{}
	{
		v = map[string]interface{}{
			"ID":   1,
			"Name": "Title",
		}
	}
	var argMap map[string]interface{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var m map[string]interface{}
		mtype := reflect.TypeOf(m)
		t := reflect.TypeOf(v)
		if !t.ConvertibleTo(mtype) {
			b.Fatal(`invalid map given, unable to convert to map[string]interface{}`)
		}
		argMap = reflect.ValueOf(v).Convert(mtype).Interface().(map[string]interface{})
	}
	if argMap == nil {
		b.Fatal("ArgMap shouldn't be nil")
	}
}
