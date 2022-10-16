// postgresql is for testing the postgresql database
package postgresql

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/silbinarywolf/sqlsw/internal/tests/testsuite"
)

// BenchmarkNamedQueryContextWithScanStructCold-12    	    1108	    990436 ns/op	    1392 B/op	      36 allocs/op
// BenchmarkNamedQueryContextWithScanStructCold-12    	    1066	   1032050 ns/op	    1392 B/op	      36 allocs/op
// BenchmarkNamedQueryContextWithScanStructCold-12    	    1066	   1032050 ns/op	    1392 B/op	      36 allocs/op
// BenchmarkNamedQueryContextWithScanStructCold-12    	    1368	    972451 ns/op	    1392 B/op	      36 allocs/op
func BenchmarkNamedQueryContextWithScanStructCold(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testsuite.NamedQueryContextWithScanStruct(b, db)

		// Remove cache
		b.StopTimer()
		testsuite.ResetCache(b, db)
		b.StartTimer()
	}
}

// BenchmarkNamedQueryContextWithScanStruct-12    	    1058	   1092404 ns/op	     792 B/op	      23 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	    1040	   1134120 ns/op	     792 B/op	      23 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	    1034	   1130647 ns/op	     792 B/op	      23 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	     999	   1163950 ns/op	     792 B/op	      23 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	    1047	   1066900 ns/op	     792 B/op	      23 allocs/op
func BenchmarkNamedQueryContextWithScanStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testsuite.NamedQueryContextWithScanStruct(b, db)
	}
}
