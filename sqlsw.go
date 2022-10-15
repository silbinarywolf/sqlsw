package sqlsw

import (
	"context"
	"database/sql"
)

type DB struct {
	// handle is a database handle from database/sql
	db *sql.DB
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	rows *sql.Rows
}

func (db *DB) NamedQueryContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	
	sqlRows, err := db.db.QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	r := &Rows{}
	r.rows = sqlRows
	return r, nil
}
