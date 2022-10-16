// postgresql is for testing the postgresql database
package postgresql

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/silbinarywolf/sqlsw/internal/tests/testsuite"
)

// standard reflect
// ----------------
// BenchmarkNamedQueryContextWithScanStruct-12    	    1058	   1092404 ns/op	     792 B/op	      23 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	    1040	   1134120 ns/op	     792 B/op	      23 allocs/op
// BenchmarkNamedQueryContextWithScanStruct-12    	    1034	   1130647 ns/op	     792 B/op	      23 allocs/op
func BenchmarkNamedQueryContextWithScanStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testsuite.NamedQueryContextWithScanStruct(b, db)
	}
}

// standard reflect
// ----------------
// BenchmarkQueryContextWithScan-12    	    1051	   1105692 ns/op	     656 B/op	      17 allocs/op
// BenchmarkQueryContextWithScan-12    	    1053	   1086366 ns/op	     656 B/op	      17 allocs/op
// BenchmarkQueryContextWithScan-12    	    1041	   1165816 ns/op	     656 B/op	      17 allocs/op
// BenchmarkQueryContextWithScan-12    	    1002	   1159123 ns/op	     656 B/op	      17 allocs/op
func BenchmarkQueryContextWithScan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testsuite.QueryContextWithScan(b, db)
	}
}
