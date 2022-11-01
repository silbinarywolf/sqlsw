//go:build go1.8
// +build go1.8

// The following environment variables, if set, will be used:
//
//   - SQLX_SQLITE_DSN
//   - SQLX_POSTGRES_DSN
//   - SQLX_MYSQL_DSN
//
// Set any of these variables to 'skip' to skip them.  Note that for MySQL,
// the string '?parseTime=True' will be appended to the DSN if it's not there
// already.
package sqlx

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func TestSliceInterface(t *testing.T) {
	type IPerson interface {
		GetFirstName() string
	}
	/*
	   type personButInterfaceCompliant struct {
	   	FirstName string `db:"first_name"`
	   	LastName  string `db:"last_name"`
	   	Email     string
	   	AddedAt   time.Time `db:"added_at"`
	   }

	   func (c *personButInterfaceCompliant) GetFirstName() string {
	   	return c.FirstName
	   }
	*/

	// note(jae): 2022-11-01
	// Improve error message returned when []InterfaceType is attempted
	// https://github.com/jmoiron/sqlx/issues/839
	RunWithSchema(defaultSchema, t, func(db *DB, t *testing.T, now string) {
		loadDefaultFixture(db, t)
		var clients []IPerson
		err := db.Select(&clients, "SELECT * FROM person ORDER BY first_name ASC")
		if err != nil {
			// note(jae): 2022-11-01
			// Not the best way to check for error correctness but it'll do for now.
			if err.Error() != "ScanSlice: must pass a pointer to a slice value, cannot be slice of interfaces as there is no way to infer what implementation to use" {
				t.Fatal(err)
			}
		}
	})
}
