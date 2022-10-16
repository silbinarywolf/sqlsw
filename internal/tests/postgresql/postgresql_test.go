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
	dataSourceName := testsuite.GetDefaultDataSourceName()
	if dataSourceName == "" {
		dataSourceName = "postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
	}
	var err error
	db, err = sqlsw.Open("postgres", dataSourceName)
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

func TestRunAllCommonTests(t *testing.T) {
	testsuite.TestRunAll(t, db)
}
