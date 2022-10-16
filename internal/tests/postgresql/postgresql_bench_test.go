// postgresql is for testing the postgresql database
package postgresql

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/silbinarywolf/sqlsw/internal/tests/testsuite"
)

// BenchmarkNamedQueryContextWithStruct-12    	     987	   1176021 ns/op	     960 B/op	      24 allocs/op
// BenchmarkNamedQueryContextWithStruct-12    	     961	   1204793 ns/op	     960 B/op	      24 allocs/op
// BenchmarkNamedQueryContextWithStruct-12    	     982	   1177490 ns/op	     960 B/op	      24 allocs/op
func BenchmarkNamedQueryContextWithScanStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testsuite.NamedQueryContextWithStruct(b, db)
	}
}
