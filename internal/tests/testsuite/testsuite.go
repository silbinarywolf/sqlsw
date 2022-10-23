// testsuite holds common tests to be used across each database driver
package testsuite

import (
	"context"
	"os"
	"testing"

	"github.com/silbinarywolf/sqlsw"
)

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func GetDefaultDataSourceName() string {
	return os.Getenv("DATABASE_URL")
}

func ResetCache(t testOrBench, db *sqlsw.DB) {
	sqlsw.TestOnlyResetCache(t, db)
}

func TestRunAll(t *testing.T, db *sqlsw.DB) {
	t.Run("NamedQueryContext with ScanStruct", func(t *testing.T) {
		NamedQueryContextWithScanStruct(t, db)
	})
}

type selectQueryStruct struct {
	ID int64 `db:"ID"`
}

func NamedQueryContextWithScanStruct(t testOrBench, db *sqlsw.DB) {
	queryRecord := selectQueryStruct{}
	queryRecord.ID = 1
	query := `select "ID" from "Operation" where "ID" = :ID`
	rows, err := db.NamedQueryContext(context.Background(), query, queryRecord)
	if err != nil {
		t.Fatalf("query failed: %s, error: %s", query, err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected a result")
	}
	var record selectQueryStruct
	if err := rows.ScanStruct(&record); err != nil {
		t.Fatal(err)
	}
	if record.ID == 0 {
		t.Fatal("ID should not be zero")
	}
}

func NamedQueryContextWithScanSliceValue(t testOrBench, db *sqlsw.DB) {
	queryRecord := selectQueryStruct{}
	queryRecord.ID = 1
	query := `select "ID" from "Operation"`
	rows, err := db.NamedQueryContext(context.Background(), query, queryRecord)
	if err != nil {
		t.Fatalf("query failed: %s, error: %s", query, err)
	}
	defer rows.Close()
	var sliceOfRecords []selectQueryStruct
	if err := rows.ScanSlice(&sliceOfRecords); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%v", sliceOfRecords)
}

func NamedQueryContextWithScanSlicePtr(t testOrBench, db *sqlsw.DB) {
	queryRecord := selectQueryStruct{}
	queryRecord.ID = 1
	query := `select "ID" from "Operation"`
	rows, err := db.NamedQueryContext(context.Background(), query, queryRecord)
	if err != nil {
		t.Fatalf("query failed: %s, error: %s", query, err)
	}
	defer rows.Close()
	var sliceOfRecords []*selectQueryStruct
	if err := rows.ScanSlice(&sliceOfRecords); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%+v", sliceOfRecords[1])
}
