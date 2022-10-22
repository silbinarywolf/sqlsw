// sqlx is a compatibility layer for sqlx
package sqlx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/silbinarywolf/sqlsw"
)

type DB struct {
	db         sqlsw.DB
	driverName string
	// allowUnknownFields maps to unsafe in sqlx
	allowUnknownFields bool
}

func Open(driverName, dataSourceName string) (*DB, error) {
	dbDriver, err := sqlsw.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	db := &DB{}
	db.driverName = driverName
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

// Unsafe returns a version of DB which will silently succeed to scan when
// columns in the SQL result have no fields in the destination struct.
// sqlx.Stmt and sqlx.Tx which are created from this DB will inherit its
// safety behavior.
func (db *DB) Unsafe() *DB {
	newDB := new(DB)
	*newDB = *db
	newDB.allowUnknownFields = true
	return newDB
}

// Rebind a query within a transaction's bindvar type.
func (db *DB) Rebind(query string) string {
	panic("TODO(jae): 2022-10-22: Implement Rebind")
	//return Rebind(BindType(tx.driverName), query)
}

// DriverName returns the driverName passed to the Open function for this DB.
func (db *DB) DriverName() string {
	return db.driverName
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return sqlsw.SQLX_DB(&db.db).QueryContext(ctx, query, args...)
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

// MustBegin starts a transaction, and panics on error.  Returns an *sqlx.Tx instead
// of an *sql.Tx.
func (db *DB) MustBegin() *Tx {
	tx, err := db.Beginx()
	if err != nil {
		panic(err)
	}
	return tx
}

// MustBeginTx starts a transaction, and panics on error.  Returns an *sqlx.Tx instead
// of an *sql.Tx.
//
// The provided context is used until the transaction is committed or rolled
// back. If the context is canceled, the sql package will roll back the
// transaction. Tx.Commit will return an error if the context provided to
// MustBeginContext is canceled.
func (db *DB) MustBeginTx(ctx context.Context, opts *sql.TxOptions) *Tx {
	tx, err := db.BeginTxx(ctx, opts)
	if err != nil {
		panic(err)
	}
	return tx
}

// Beginx begins a transaction and returns an *sqlx.Tx instead of an *sql.Tx.
func (db *DB) Beginx() (*Tx, error) {
	panic("TODO(jae): 2022-10-22: Implement db.Beginx")
	/* tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{
		underlying: tx,
		driverName: db.driverName,
		unsafe:     db.unsafe,
		//Mapper: db.Mapper
	}, err */
}

// BeginTxx begins a transaction and returns an *sqlx.Tx instead of an
// *sql.Tx.
//
// The provided context is used until the transaction is committed or rolled
// back. If the context is canceled, the sql package will roll back the
// transaction. Tx.Commit will return an error if the context provided to
// BeginxContext is canceled.
func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	panic("TODO(jae): 2022-10-22: Implement BeginTxx")
	/* tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{
		tx:         tx,
		driverName: db.DriverName(),
		unsafe:     db.db,
		//Mapper: db.Mapper
	}, err */
}

// Exec executes a named statement using the struct passed.
// Any named placeholder parameters are replaced with fields from arg.
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return sqlsw.SQLX_DB(&db.db).Exec(query, args...)
}

// QueryxContext queries the database and returns an *sqlx.Rows.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	sqlRows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	rowsUnderlying := sqlsw.SQLX_NewRows(sqlRows, &db.db)
	return &Rows{
		rows:               *rowsUnderlying,
		allowUnknownFields: db.allowUnknownFields,
		// note(jae): 2022-10-22
		// Not supporting Mapper, at least at time of writing
		// Mapper: db.Mapper
	}, err
}

// BindNamed binds a query using the DB driver's bindvar type.
func (*DB) BindNamed(query string, structOrMapArg interface{}) (string, []interface{}, error) {
	panic("TODO(jae): 2022-10-22: Implement BindNamed")
	// return bindNamedMapper(BindType(db.driverName), query, arg, db.Mapper)
}

// SelectContext using this DB.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	panic("TODO(jae): 2022-10-22: Implement SelectContext")
	//return SelectContext(ctx, db, dest, query, args...)
}

// Select using this DB.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return db.SelectContext(context.Background(), dest, query, args...)
}

// GetContext using this DB.
// Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	panic("TODO(Jae): 2022-10-22: Support db.GetContext")
	// return GetContext(ctx, db, dest, query, args...)
}

// Get using this DB.
// Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return db.GetContext(context.Background(), dest, query, args...)
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

func (rows *Rows) Err() error {
	// todo(jae): 2022-10-22
	// Probably expose this in sqlsw.Rows?
	return sqlsw.SQLX_Rows(&rows.rows).Err()
}

func (rows *Rows) Columns() ([]string, error) {
	// todo(jae): 2022-10-22
	// Probably expose this in sqlsw.Rows?
	return sqlsw.SQLX_Rows(&rows.rows).Columns()
}

func (rows *Rows) Scan(dest ...interface{}) error {
	return sqlsw.SQLX_Rows(&rows.rows).Scan(dest...)
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

// Tx is an in-progress database transaction.
type Tx struct {
	underlying sqlsw.Tx
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	return tx.underlying.Commit()
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback() error {
	return tx.underlying.Rollback()
}

// MustExec executes a query that doesn't return rows.
// For example: an INSERT and UPDATE.
func (tx *Tx) MustExec(query string, args ...interface{}) sql.Result {
	sqlResult, err := sqlsw.SQLX_Tx(&tx.underlying).Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return sqlResult
}

// Rebind a query within a transaction's bindvar type.
func (tx *Tx) Rebind(query string) string {
	panic("TODO(jae): 2022-10-22: Implement tx.Rebind")
	//return Rebind(BindType(tx.driverName), query)
}

// NamedStmtContext returns a version of the prepared statement which runs
// within a transaction.
func (tx *Tx) NamedStmtContext(ctx context.Context, stmt *NamedStmt) *NamedStmt {
	panic("TODO(jae): 2022-10-22: Implement tx.NamedStmtContext")
	/* return &NamedStmt{
		QueryString: stmt.QueryString,
		Params:      stmt.Params,
		Stmt:        tx.StmtxContext(ctx, stmt.Stmt),
	} */
}

type rowsi interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(...interface{}) error
}

func StructScan(rows rowsi, ptrValue interface{}) error {
	panic("TODO(jae): 2022-10-22: handle StructScan")
	switch rows := rows.(type) {
	case *Rows:
		return rows.StructScan(ptrValue)
	case *sql.Rows:
		/* rowsUnderlying := sqlsw.SQLX_NewRows(rows, &db.db)
		rows := &Rows{
			rows: *rowsUnderlying,
			// note(jae): 2022-10-22
			// Not supporting Mapper, at least at time of writing
			// Mapper: db.Mapper
		}
		return rows.StructScan(ptrValue) */
	}
	return errors.New("unable to execute StructScan")
}
