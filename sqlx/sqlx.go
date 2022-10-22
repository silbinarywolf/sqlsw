// sqlx is a compatibility layer for sqlx
package sqlx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/silbinarywolf/sqlsw"
)

type DB struct {
	db sqlsw.DB
	// allowUnknownFields maps to unsafe in sqlx
	allowUnknownFields bool
}

// Tx is an in-progress database transaction.
type Tx struct {
	tx sqlsw.Tx
}

func Open(driverName, dataSourceName string) (*DB, error) {
	dbDriver, err := sqlsw.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	db := &DB{}
	db.db = *dbDriver
	return db, err
}

// PrepareNamedContext creates a prepared statement for later queries or executions.
func (db *DB) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	namedStmtUnderlying, err := db.db.NamedPrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	stmt := &NamedStmt{}
	stmt.namedStmt = *namedStmtUnderlying
	return stmt, nil
}

// Execer is an interface used by MustExec and LoadFile
type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Queryer is an interface used by Get and Select
type Queryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Queryx(query string, args ...interface{}) (*Rows, error)
	QueryRowx(query string, args ...interface{}) *Row
}

// Binder is an interface for something which can bind queries (Tx, DB)
type binder interface {
	DriverName() string
	Rebind(string) string
	BindNamed(string, interface{}) (string, []interface{}, error)
}

// Ext is a union interface which can bind, query, and exec, used by
// NamedQuery and NamedExec.
type Ext interface {
	binder
	Queryer
	Execer
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.db.DB().QueryContext(ctx, query, args...)
}

// QueryRowxContext queries the database and returns an *sqlx.Row.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row {
	panic("TODO(jae): 2022-10-22: Support QueryRowxContext")
	/* rows, err := db.QueryContext(ctx, query, args...)
	return &Row{
		rows:               rows,
		err:                err,
		allowUnknownFields: db.allowUnknownFields,
		// note(jae): 2022-10-22
		// Not supporting Mapper, at least at time of writing
		//, Mapper: db.Mapper
	} */
}

// QueryxContext queries the database and returns an *sqlx.Rows.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	sqlRows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	rowsUnderlying := sqlsw.NewRows(sqlRows, &db.db)
	return &Rows{
		rows:               *rowsUnderlying,
		allowUnknownFields: db.allowUnknownFields,
		// note(jae): 2022-10-22
		// Not supporting Mapper, at least at time of writing
		// Mapper: db.Mapper
	}, err
}

func (*DB) BindNamed(query string, structOrMapArg interface{}) (string, []interface{}, error) {
	panic("TODO(jae): 2022-10-22: Support BindNamed")
	// return bindNamedMapper(BindType(db.driverName), query, arg, db.Mapper)
}

// GetContext using this DB.
// Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return GetContext(ctx, db, dest, query, args...)
}

type NamedStmt struct {
	namedStmt sqlsw.NamedStmt
}

func (n *NamedStmt) QueryxContext(ctx context.Context, structOrMapArg interface{}) (*Rows, error) {
	panic("TODO(jae): 2022-10-22: Support QueryxContext")
}

// SelectContext using this NamedStmt
// Any named placeholder parameters are replaced with fields from structOrMapArg.
func (n *NamedStmt) SelectContext(ctx context.Context, dest interface{}, structOrMapArg interface{}) error {
	rows, err := n.QueryxContext(ctx, structOrMapArg)
	if err != nil {
		return err
	}
	// if something happens here, we want to make sure the rows are Closed
	defer rows.Close()
	return errors.New("TODO(jae): 2022-10-22: Implement SelectContext")
	// return scanAll(rows, dest, false)
}

// ExecContext executes a named statement using the struct passed.
// Any named placeholder parameters are replaced with fields from arg.
func (n *NamedStmt) ExecContext(ctx context.Context, structOrMapArg interface{}) (sql.Result, error) {
	return *new(sql.Result), errors.New("TODO(jae): 2022-10-22: Implement ExecContext")
	/* args, err := bindAnyArgs(n.Params, structOrMapArg, n.Stmt.Mapper)
	if err != nil {
		return *new(sql.Result), err
	}
	return n.namedStmt.Stmt().ExecContext(ctx, args...) */
}

// Close closes the statement.
func (stmt *NamedStmt) Close() error {
	return stmt.namedStmt.Close()
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	rows               sqlsw.Rows
	allowUnknownFields bool
}

func (rows *Rows) Next() bool {
	return rows.rows.Next()
}

func (rows *Rows) Close() error {
	return rows.rows.Close()
}

// StructScan copies the columns in the current row into the given struct.
func (rows *Rows) StructScan(ptrValue interface{}) error {
	return rows.rows.ScanStruct(ptrValue)
}

// Row is the result of calling QueryRow to select a single row.
type Row struct {
	row                sqlsw.Row
	allowUnknownFields bool
}

// GetContext does a QueryRow using the provided Queryer, and scans the
// resulting row to dest.  If dest is scannable, the result must only have one
// column. Otherwise, StructScan is used.  Get will return sql.ErrNoRows like
// row.Scan would. Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func GetContext(ctx context.Context, q QueryerContext, dest interface{}, query string, args ...interface{}) error {
	panic("TODO(Jae): 202-10-22: Support QueryerContext")
	//r := q.QueryRowxContext(ctx, query, args...)
	//return r.scanAny(dest, false)
}
