// postgresql is for testing the postgresql database
package postgresql

import (
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/silbinarywolf/sqlsw"
	"github.com/silbinarywolf/sqlsw/internal/tests/testsuite"
)

var (
	db *sqlsw.DB
)

func testMain(m *testing.M) error {
	var err error
	db, err = newDB()
	if err != nil {
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	if err := testMain(m); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func newDB() (*sqlsw.DB, error) {
	dataSourceName := testsuite.GetDefaultDataSourceName()
	if dataSourceName == "" {
		dataSourceName = "postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
	}
	var err error
	db, err = sqlsw.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestNamedQueryContextWithScanStruct(t *testing.T) {
	testsuite.NamedQueryContextWithScanStruct(t, db)
}

func TestNamedQueryContextWithScanSliceValue(t *testing.T) {
	testsuite.NamedQueryContextWithScanSliceValue(t, db)
}

func TestNamedQueryContextWithScanSlicePtr(t *testing.T) {
	testsuite.NamedQueryContextWithScanSlicePtr(t, db)
}
