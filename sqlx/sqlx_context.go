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
