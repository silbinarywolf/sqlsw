// postgresql is for testing the postgresql database
package postgresql

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/silbinarywolf/sqlsw/internal/tests/testsuite"
)

// standard reflect
// ----------------
// BenchmarkNamedQueryContextWithScanStruct-12    	 1080	   1054847 ns/op	     960 B/op	      24 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	 1094	   1086777 ns/op	     960 B/op	      24 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	 1047	   1087898 ns/op	     960 B/op	      24 allocs/op
//
// reflect "github.com/goccy/go-reflect"
// -------------------------------------
// BenchmarkNamedQueryContextWithScanStruct-12    	 1092	   1027058 ns/op	     960 B/op	      24 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	 1052	   1057300 ns/op	     960 B/op	      24 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	 1131	   1039205 ns/op	     960 B/op	      24 allocs/op
func BenchmarkNamedQueryContextWithScanStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testsuite.NamedQueryContextWithScanStruct(b, db)
	}
}
