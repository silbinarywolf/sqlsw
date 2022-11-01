// sqlx is a compatibility layer for sqlx
package sqlx

import (
	"context"
	"database/sql"
)

// ExecerContext is an interface used by MustExecContext and LoadFileContext
type ExecerContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// QueryerContext is an interface used by GetContext and SelectContext
type QueryerContext interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row
}

// ConnectContext to a database and verify with a ping.
func ConnectContext(ctx context.Context, driverName, dataSourceName string) (*DB, error) {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return nil, err
	// note(jae): 2022-11-01
	// Original SQLX code does not close the DB connectin on a Ping error so we do
	// that. Can revert if this causes backwards compat issues for users.
	//db, err := Open(driverName, dataSourceName)
	//if err != nil {
	//	return nil, err
	//}
	//err = db.PingContext(ctx)
	//return db, err
}
