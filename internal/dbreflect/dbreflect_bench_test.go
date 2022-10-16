package dbreflect

import (
	"testing"
)

type simpleStruct struct {
	ID    int64  `db:"ID"`
	Title string `db:"Title"`
}

type commonStructForNesting struct {
	ID int64 `db:"ID"`
}

type nestedStruct struct {
	commonStructForNesting
	Title string `db:"Title"`
}

func BenchmarkGetStruct(b *testing.B) {
	m := ReflectModule{}
	v := nestedStruct{}
	reflectValue := ValueOf(v)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		structInfo, err := m.GetStruct(TypeOf(v))
		if err != nil {
			b.Fatal(err)
		}
		if len(structInfo.fields) == 0 {
			b.Fatal("no fields found on struct")
		}
		if _, ok := structInfo.fields[0].Interface(reflectValue).(int64); !ok {
			b.Fatal("fields[0] should be int64")
		}
		if _, ok := structInfo.fields[1].Interface(reflectValue).(string); !ok {
			b.Fatal("fields[1] should be string")
		}
	}
}
