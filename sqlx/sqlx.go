// sqlx is a compatibility layer for sqlx
package sqlx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/silbinarywolf/sqlsw"
)

type DB struct {
	db         sqlsw.DB
	driverName string
	metadataInfo
}

// NewDb returns a new sqlx DB wrapper for a pre-existing *sql.DB.  The
// driverName of the original database is required for named query support.
//
// Unlike the original sqlx library, this version can crash if you haven't defined the bind types
// for your driver yet.
func NewDb(db *sql.DB, driverName string) *DB {
	dbSw, err := sqlsw.SQLX_CompatNewDB(db, driverName)
	if err != nil {
		panic(err)
	}
	dbR := &DB{}
	dbR.driverName = driverName
	dbR.db = *dbSw
	return dbR
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

// Connect to a database and verify with a ping.
func Connect(driverName, dataSourceName string) (*DB, error) {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) Ping() error {
	return db.db.PingContext(context.Background())
}

func (db *DB) PingContext(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

// PrepareNamedContext creates a prepared statement for later queries or executions.
func (db *DB) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	namedStmtUnderlying, err := db.db.NamedPrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	stmt := &NamedStmt{}
	stmt.namedStmt = *namedStmtUnderlying
	stmt.unsafe = db.unsafe
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
	newDB.unsafe = true
	return newDB
}

func (db *DB) isUnsafe() bool {
	return db.unsafe
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

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return sqlsw.SQLX_DB(&db.db).QueryRowContext(ctx, query, args...)
}

// QueryRowx queries the database and returns an *sqlx.Row.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) QueryRowx(query string, args ...interface{}) *Row {
	return db.QueryRowxContext(context.Background(), query, args...)
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

// Query executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.
//
// Query uses context.Background internally; to specify the context, use
// QueryContext.
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

// QueryxContext queries the database and returns an *sqlx.Rows, typically a SELECT.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) Queryx(query string, args ...interface{}) (*Rows, error) {
	return db.QueryxContext(context.Background(), query, args...)
}

// QueryxContext queries the database and returns an *sqlx.Rows, typically a SELECT.
// Any placeholder parameters are replaced with supplied args.
func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	sqlRows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	rowsUnderlying := sqlsw.SQLX_NewRows(sqlRows, &db.db)
	return newRows(*rowsUnderlying, db.metadataInfo), nil
}

// ExecContext executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return sqlsw.SQLX_DB(&db.db).ExecContext(ctx, query, args...)
}

// MustExec using this DB.
// Any named placeholder parameters are replaced with fields from arg.
func (db *DB) MustExec(query string, args ...interface{}) sql.Result {
	r, err := db.ExecContext(context.Background(), query, args...)
	if err != nil {
		panic(err)
	}
	return r
}

// NamedExecContext using this DB.
// Any named placeholder parameters are replaced with fields from arg.
func (db *DB) NamedExecContext(ctx context.Context, query string, structOrMapArg interface{}) (sql.Result, error) {
	return db.db.NamedExecContext(ctx, query, structOrMapArg)
}

// NamedExec using this DB.
// Any named placeholder parameters are replaced with fields from arg.
func (db *DB) NamedExec(query string, structOrMapArg interface{}) (sql.Result, error) {
	return db.NamedExecContext(context.Background(), query, structOrMapArg)
}

func (db *DB) NamedQueryContext(ctx context.Context, query string, structOrMapArg interface{}) (*Rows, error) {
	rowsUnderlying, err := db.db.NamedQueryContext(ctx, query, structOrMapArg)
	if err != nil {
		return nil, err
	}
	return newRows(*rowsUnderlying, db.metadataInfo), nil
}

func (db *DB) NamedQuery(query string, structOrMapArg interface{}) (*Rows, error) {
	return db.NamedQueryContext(context.Background(), query, structOrMapArg)
}

// PrepareNamed returns an sqlx.NamedStmt
func (db *DB) PrepareNamed(query string) (*NamedStmt, error) {
	return db.PrepareNamedContext(context.Background(), query)
}

// BindNamed binds a query using the DB driver's bindvar type.
func (*DB) BindNamed(query string, structOrMapArg interface{}) (string, []interface{}, error) {
	panic("TODO(jae): 2022-10-22: Implement BindNamed")
	// return bindNamedMapper(BindType(db.driverName), query, arg, db.Mapper)
}

// PreparexContext returns an sqlx.Stmt instead of a sql.Stmt.
//
// The provided context is used for the preparation of the statement, not for
// the execution of the statement.
func (db *DB) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := sqlsw.SQLX_DB(&db.db).PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return newStmt(stmt, db.metadataInfo), nil
}

// Preparex returns an sqlx.Stmt instead of a sql.Stmt.
func (db *DB) Preparex(query string) (*Stmt, error) {
	return db.PreparexContext(context.Background(), query)
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
	// unsafe is true when unknown fields are allowed
	unsafe bool
}

func (n *NamedStmt) Queryx(structOrMapArg interface{}) (*Rows, error) {
	return n.QueryxContext(context.Background(), structOrMapArg)
}

func (n *NamedStmt) QueryxContext(ctx context.Context, structOrMapArg interface{}) (*Rows, error) {
	panic("TODO(jae): 2022-10-22: Support QueryxContext")
}

func (n *NamedStmt) Unsafe() *NamedStmt {
	newNamedStmt := new(NamedStmt)
	*newNamedStmt = *n
	newNamedStmt.unsafe = true
	return newNamedStmt
}

// Select using this NamedStmt
// Any named placeholder parameters are replaced with fields from structOrMapArg.
func (n *NamedStmt) Select(dest interface{}, structOrMapArg interface{}) error {
	return n.SelectContext(context.Background(), dest, structOrMapArg)
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
	return nil, errors.New("TODO(jae): 2022-10-22: Implement ExecContext")
	/* args, err := bindAnyArgs(n.Params, structOrMapArg, n.Stmt.Mapper)
	if err != nil {
		return nil, err
		// note(jae): 2022-10-22
		// SQLX returns *new(sql.Result) but thats then returning
		// a newed interface. Probably shouldn't do that.
		// return *new(sql.Result), err
	}
	return n.namedStmt.Stmt().ExecContext(ctx, args...) */
}

// func (stmt *NamedStmt) Stmt() *sql.Stmt {
//	return sqlsw.SQLX_NamedStmt(&stmt.namedStmt)
//}

// Close closes the statement.
func (stmt *NamedStmt) Close() error {
	return stmt.namedStmt.Close()
}

type metadataInfo struct {
	// unsafe is true when unknown fields are allowed
	unsafe bool
	// note(jae): 2022-10-22
	// Not supporting Mapper, at least at time of writing
	// Mapper: db.Mapper
}

//func (meta *metadataInfo) isUnsafe() bool {
//	return meta.unsafe
//}

type metadatai interface {
	isUnsafe() bool
}

type Stmt struct {
	stmt *sql.Stmt
	metadataInfo
}

func newStmt(stmt *sql.Stmt, metadata metadataInfo) *Stmt {
	newStmt := &Stmt{}
	newStmt.stmt = stmt
	newStmt.metadataInfo = metadata
	return newStmt
}

// SelectContext using the prepared statement.
// Any placeholder parameters are replaced with supplied args.
func (stmt *Stmt) SelectContext(ctx context.Context, dest interface{}, args ...interface{}) error {
	panic("TODO(jae): 2022-10-22: Support stmt.SelectContext")
	// return SelectContext(ctx, db, dest, query, args...)
}

// Select using the prepared statement.
// Any placeholder parameters are replaced with supplied args.
func (stmt *Stmt) Select(dest interface{}, args ...interface{}) error {
	return stmt.SelectContext(context.Background(), dest, args...)
}

// GetContext using this statement.
// Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func (stmt *Stmt) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	panic("TODO(jae): 2022-10-22: Support stmt.GetContext")
	// return GetContext(ctx, db, dest, query, args...)
}

// Get using this statement.
// Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func (stmt *Stmt) Get(dest interface{}, query string, args ...interface{}) error {
	return stmt.GetContext(context.Background(), dest, query, args...)
}

// QueryRowx queries the database and returns an *sqlx.Row.
// Any placeholder parameters are replaced with supplied args.
func (stmt *Stmt) QueryRowx(query string, args ...interface{}) *Row {
	return stmt.QueryRowxContext(context.Background(), query, args...)
}

// QueryRowxContext queries the database and returns an *sqlx.Row.
// Any placeholder parameters are replaced with supplied args.
func (stmt *Stmt) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row {
	panic("todo(jae): 2022-10-22: implement stmt.QueryRowxContext")
}

func (stmt *Stmt) Queryx(query string, args ...interface{}) (*Rows, error) {
	rows, err := stmt.stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	return newRows(*sqlsw.SQLX_NewRows(rows, nil), stmt.metadataInfo), nil
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	rows sqlsw.Rows
	metadataInfo
}

func newRows(rows sqlsw.Rows, metadata metadataInfo) *Rows {
	newRows := &Rows{}
	newRows.rows = rows
	newRows.metadataInfo = metadata
	return newRows
}

// SliceScan using Rows
func (rows *Rows) SliceScan() ([]interface{}, error) {
	panic("todo(jae): 2022-10-22: Implement rows.SliceScan")
	// return SliceScan(r)
}

// MapScan scans a single Row into the dest map[string]interface{}.
func (rows *Rows) MapScan(dest map[string]interface{}) error {
	panic("todo(jae): 2022-10-22: Implement rows.MapScan")
	// return MapScan(r)
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
	row sqlsw.Row
	// unsafe is true when unknown fields are allowed
	unsafe bool
}

// Scan copies the columns in the current row into the values pointed
// at by dest. The number of values in dest must be the same as the
// number of columns in Rows.
func (row *Row) Scan(args ...interface{}) error {
	return sqlsw.SQLX_Rows_From_Row(&row.row).Scan(args...)
}

// StructScan copies the columns in the current row into the given struct.
func (rows *Row) StructScan(ptrValue interface{}) error {
	return rows.row.ScanStruct(ptrValue)
}

// SliceScan using this Row
func (row *Row) SliceScan() ([]interface{}, error) {
	panic("todo(jae): 2022-10-22: Implement row.SliceScan")
	// return SliceScan(r)
}

// MapScan scans a single Row into the dest map[string]interface{}.
func (rows *Row) MapScan(dest map[string]interface{}) error {
	panic("todo(jae): 2022-10-22: Implement row.MapScan")
	// return MapScan(r)
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
	// unsafe is true when unknown fields are allowed
	unsafe bool
}

func (tx *Tx) isUnsafe() bool {
	return tx.unsafe
}

// StmtContext returns a transaction-specific prepared statement from
// an existing statement.
//
// Example:
//
//	updateMoney, err := db.Prepare("UPDATE balance SET money=money+? WHERE id=?")
//	...
//	tx, err := db.Begin()
//	...
//	res, err := tx.StmtContext(ctx, updateMoney).Exec(123.45, 98293203)
//
// The provided context is used for the preparation of the statement, not for the
// execution of the statement.
//
// The returned statement operates within the transaction and will be closed
// when the transaction has been committed or rolled back.
func (tx *Tx) StmtContext(ctx context.Context, stmt *sql.Stmt) *sql.Stmt {
	return sqlsw.SQLX_Tx(&tx.underlying).StmtContext(ctx, stmt)
}

// Stmt returns a transaction-specific prepared statement from
// an existing statement.
func (tx *Tx) Stmt(stmt *sql.Stmt) *sql.Stmt {
	return tx.StmtContext(context.Background(), stmt)
}

// Stmtx returns a version of the prepared statement which runs within a transaction.  Provided
// stmt can be either *sql.Stmt or *sqlx.Stmt.
func (tx *Tx) Stmtx(stmt interface{}) *Stmt {
	var (
		s *sql.Stmt
	)
	switch v := stmt.(type) {
	case Stmt:
		s = v.stmt
	case *Stmt:
		s = v.stmt
	case *sql.Stmt:
		s = v
	default:
		panic(fmt.Sprintf("non-statement type %v passed to Stmtx", reflect.ValueOf(stmt).Type()))
	}
	return &Stmt{
		stmt: tx.Stmt(s),
		// note(jae): 2022-10-22
		// this isn't set in SQLX, so we're keeping this behaviour.
		// unsafe: false,
		// note(jae): 2022-10-22
		// Not supporting mapper for the time-being
		// Mapper: tx.Mapper
	}
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

// determine if any of our extensions are unsafe
func isUnsafe(i interface{}) bool {
	switch v := i.(type) {
	case Row:
		return v.unsafe
	case *Row:
		return v.unsafe
	case Rows:
		return v.unsafe
	case *Rows:
		return v.unsafe
	case NamedStmt:
		return v.unsafe

		//return v.Stmt.unsafe
	case *NamedStmt:
		return v.unsafe
		// note(jae): 2022-10-22
		// Original SQLX checked for this:
		// return v.Stmt.unsafe
	case Stmt:
		return v.unsafe
	case *Stmt:
		return v.unsafe
	// todo(jae): 2022-10-22
	// implement qStmt support if needed
	/* case qStmt:
		return v.unsafe
	case *qStmt:
		return v.unsafe */
	case DB:
		return v.unsafe
	case *DB:
		return v.unsafe
	case Tx:
		return v.unsafe
	case *Tx:
		return v.unsafe
	case sql.Rows, *sql.Rows:
		return false
	default:
		return false
	}
}
