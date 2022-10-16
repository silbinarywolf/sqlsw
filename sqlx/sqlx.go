// sqlx is a compatibility layer for sqlx
package sqlx

import (
	"context"
	"database/sql"

	"github.com/silbinarywolf/sqlsw"
)

type DB struct {
	dbWrapper
}

type dbWrapper struct {
	sqlsw.DB
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	rowsEmbed
}

// rowsEmbed exists to add another layer of indirection so a user can't change
// the pointer to Rows it's holding
type rowsEmbed struct {
	*sql.Rows
}

func Open(driverName, dataSourceName string) (*DB, error) {
	dbDriver, err := sqlsw.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	db := &DB{}
	db.dbWrapper.DB = *dbDriver
	return db, nil
}

func (db *DB) NamedQueryContext(ctx context.Context, query string, args interface{}) (*Rows, error) {
	rows, err := db.dbWrapper.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	r := &Rows{}
	r.rowsEmbed.Rows = rows
	return r, nil
}
