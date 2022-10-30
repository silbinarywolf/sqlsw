// sqlx is a compatibility layer for sqlx
package sqlx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/silbinarywolf/sqlsw"
	"github.com/silbinarywolf/sqlsw/internal/sqlxcompat"
)

type DB struct {
	db sqlsw.DB
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
	dbR.bindType = int(sqlsw.SQLX_GetBindType(&dbR.db))
	return dbR
}

func Open(driverName, dataSourceName string) (*DB, error) {
	dbDriver, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	dbSw, err := sqlsw.SQLX_CompatNewDB(dbDriver, driverName)
	if err != nil {
		return nil, err
	}
	db := &DB{}
	db.driverName = driverName
	db.db = *dbSw
	db.bindType = int(sqlsw.SQLX_GetBindType(&db.db))
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

func newNamedStmt(namedStmt sqlsw.NamedStmt, metadata metadataInfo) *NamedStmt {
	nstmt := &NamedStmt{}
	nstmt.namedStmt = namedStmt
	nstmt.metadataInfo = metadata
	return nstmt
}

// PrepareNamedContext creates a prepared statement for later queries or executions.
func (db *DB) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	namedStmtUnderlying, err := db.db.NamedPrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return newNamedStmt(*namedStmtUnderlying, db.metadataInfo), nil
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
	sqlsw.SQLX_Unsafe(sqlxcompat.Use{}, &newDB.db)
	return newDB
}

func (db *DB) isUnsafe() bool {
	return sqlsw.SQLX_IsUnsafe(sqlxcompat.Use{}, &db.db)
}

func (db *DB) testDisableUnsafe() {
	sqlsw.SQLX_TestDisableUnsafe(sqlxcompat.Use{}, &db.db)
}

// Rebind a query within a transaction's bindvar type.
func (db *DB) Rebind(query string) string {
	return Rebind(db.bindType, query)
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

func newTx(tx sqlsw.Tx, metadata metadataInfo) *Tx {
	t := &Tx{}
	t.underlying = tx
	t.metadataInfo = metadata
	return t
}

// Begin starts a transaction. The default isolation level is dependent on
// the driver.
//
// Begin uses context.Background internally; to specify the context, use
// BeginTx.
func (db *DB) Begin() (*sql.Tx, error) {
	return sqlsw.SQLX_DB(&db.db).Begin()
}

// BeginTx starts a transaction.
//
// The provided context is used until the transaction is committed or rolled back.
// If the context is canceled, the sql package will roll back
// the transaction. Tx.Commit will return an error if the context provided to
// BeginTx is canceled.
//
// The provided TxOptions is optional and may be nil if defaults should be used.
// If a non-default isolation level is used that the driver doesn't support,
// an error will be returned.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return sqlsw.SQLX_DB(&db.db).BeginTx(ctx, opts)
}

// Beginx begins a transaction and returns an *sqlx.Tx instead of an *sql.Tx.
func (db *DB) Beginx() (*Tx, error) {
	sqlswTx, err := db.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return newTx(*sqlswTx, db.metadataInfo), nil
}

// BeginTxx begins a transaction and returns an *sqlx.Tx instead of an
// *sql.Tx.
//
// The provided context is used until the transaction is committed or rolled
// back. If the context is canceled, the sql package will roll back the
// transaction. Tx.Commit will return an error if the context provided to
// BeginxContext is canceled.
func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	sqlswTx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTx(*sqlswTx, db.metadataInfo), nil
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
	rowsUnderlying := sqlsw.SQLX_NewRows(sqlRows, &db.db, &db.db)
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
	sqlRows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return err
	}
	rows := sqlsw.SQLX_NewRows(sqlRows, &db.db, &db.db)
	defer rows.Close()
	if err := rows.ScanSlice(dest); err != nil {
		return err
	}
	return rows.Err()
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
	sqlRows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return err
	}
	rows := sqlsw.SQLX_NewRows(sqlRows, &db.db, &db.db)
	defer rows.Close()
	if !rows.Next() {
		err = rows.Err()
		if err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := rows.ScanStruct(dest); err != nil {
		return err
	}
	return rows.Err()
}

// Get using this DB.
// Any placeholder parameters are replaced with supplied args.
// An error is returned if the result set is empty.
func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return db.GetContext(context.Background(), dest, query, args...)
}

type NamedStmt struct {
	namedStmt sqlsw.NamedStmt
	metadataInfo
}

func (n *NamedStmt) Queryx(structOrMapArg interface{}) (*Rows, error) {
	return n.QueryxContext(context.Background(), structOrMapArg)
}

func (n *NamedStmt) QueryxContext(ctx context.Context, structOrMapArg interface{}) (*Rows, error) {
	sqlswRows, err := n.namedStmt.NamedQueryContext(ctx, structOrMapArg)
	if err != nil {
		return nil, err
	}
	return newRows(*sqlswRows, n.metadataInfo), nil
}

func (n *NamedStmt) Unsafe() *NamedStmt {
	newNamedStmt := new(NamedStmt)
	*newNamedStmt = *n
	sqlsw.SQLX_Unsafe(sqlxcompat.Use{}, &newNamedStmt.namedStmt)
	// note(jae): 2022-10-29
	// Bug in SQLX makes the underlying named statement unsafe so
	// we retain that behaviour for backwards compat.
	//
	// See "TestMissingNames" in sqlx_test.go
	sqlsw.SQLX_Unsafe(sqlxcompat.Use{}, &n.namedStmt)
	return newNamedStmt
}

func (n *NamedStmt) isUnsafe() bool {
	return sqlsw.SQLX_IsUnsafe(sqlxcompat.Use{}, &n.namedStmt)
}

// Select using this NamedStmt
// Any named placeholder parameters are replaced with fields from structOrMapArg.
func (n *NamedStmt) Select(dest interface{}, structOrMapArg interface{}) error {
	return n.SelectContext(context.Background(), dest, structOrMapArg)
}

// SelectContext using this NamedStmt
// Any named placeholder parameters are replaced with fields from structOrMapArg.
func (n *NamedStmt) SelectContext(ctx context.Context, dest interface{}, structOrMapArg interface{}) error {
	rows, err := n.namedStmt.NamedQueryContext(ctx, structOrMapArg)
	if err != nil {
		return err
	}
	defer rows.Close()
	if err := rows.ScanSlice(dest); err != nil {
		return err
	}
	return rows.Err()

	// note(jae): 2022-10-29
	// Copied from SQLX
	/* rows, err := n.QueryxContext(ctx, structOrMapArg)
	if err != nil {
		return err
	}
	// if something happens here, we want to make sure the rows are Closed
	defer rows.Close()
	return errors.New("TODO(jae): 2022-10-22: Implement namedStmt.SelectContext") */
	// return scanAll(rows, dest, false)
}

// ExecContext executes a named statement using the struct passed.
// Any named placeholder parameters are replaced with fields from arg.
func (n *NamedStmt) ExecContext(ctx context.Context, structOrMapArg interface{}) (sql.Result, error) {
	sqlResult, err := n.namedStmt.NamedExecContext(ctx, structOrMapArg)
	if err != nil {
		// note(jae): 2022-10-22
		// SQLX returns *new(sql.Result) but thats then returning
		// a newed interface. Probably shouldn't do that.
		// return *new(sql.Result), err
		return nil, err
	}
	return sqlResult, nil
}

// func (stmt *NamedStmt) Stmt() *sql.Stmt {
//	return sqlsw.SQLX_NamedStmt(&stmt.namedStmt)
//}

// Close closes the statement.
func (stmt *NamedStmt) Close() error {
	return stmt.namedStmt.Close()
}

type metadataInfo struct {
	// driverName is the driver being used
	driverName string
	bindType   int
	// note(jae): 2022-10-22
	// Not supporting Mapper, at least at time of writing
	// Mapper: db.Mapper
}

//func (meta *metadataInfo) isUnsafe() bool {
//	return meta.unsafe
//}

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

func (stmt *Stmt) isUnsafe() bool {
	// note(jae): 2022-10-29
	// No underlying, so default to false
	return false
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
	rowsSw := sqlsw.SQLX_NewRows(
		rows,
		sqlsw.SQLX_DefaultOptionsObject(sqlxcompat.Use{}),
		sqlsw.SQLX_DefaultCacheObject(sqlxcompat.Use{}),
	)
	return newRows(*rowsSw, stmt.metadataInfo), nil
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

func (rows *Rows) isUnsafe() bool {
	return sqlsw.SQLX_IsUnsafe(sqlxcompat.Use{}, &rows.rows)
}

// SliceScan a row, returning a []interface{} with values similar to MapScan.
// This function is primarily intended for use where the number of columns
// is not known.  Because you can pass an []interface{} directly to Scan,
// it's recommended that you do that as it will not have to allocate new
// slices per row.
func (rows *Rows) SliceScan() ([]interface{}, error) {
	return SliceScan(sqlsw.SQLX_Rows(&rows.rows))
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
}

func (row *Row) isUnsafe() bool {
	return sqlsw.SQLX_IsUnsafe(sqlxcompat.Use{}, &row.row)
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
	metadataInfo
}

func (tx *Tx) isUnsafe() bool {
	return sqlsw.SQLX_IsUnsafe(sqlxcompat.Use{}, &tx.underlying)
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

// NamedStmtContext returns a version of the prepared statement which runs
// within a transaction.
func (tx *Tx) NamedStmtContext(ctx context.Context, nstmt *NamedStmt) *NamedStmt {
	sqlswNamedStmt := tx.underlying.NamedStmtContext(ctx, &nstmt.namedStmt)
	return newNamedStmt(*sqlswNamedStmt, tx.metadataInfo)
}

type rowsi interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(...interface{}) error
}

func StructScan(rows rowsi, ptrValue interface{}) error {
	switch rows := rows.(type) {
	case *Rows:
		return rows.StructScan(ptrValue)
	case *sql.Rows:
		rowsSw := sqlsw.SQLX_NewRows(
			rows,
			sqlsw.SQLX_DefaultOptionsObject(sqlxcompat.Use{}),
			sqlsw.SQLX_DefaultCacheObject(sqlxcompat.Use{}),
		)
		rowsX := &Rows{
			rows: *rowsSw,
			// note(jae): 2022-10-22
			// Not supporting Mapper, at least at time of writing
			// Mapper: db.Mapper
		}
		return rowsX.StructScan(ptrValue)
	}
	return errors.New("unable to execute StructScan")
}

// determine if any of our extensions are unsafe
func isUnsafe(i interface{}) bool {
	// note(jae): 2022-10-29
	// a lot of these used to be "v.unsafe" but in our SQLX version
	// we do `v.isUnsafe()``
	switch v := i.(type) {
	case Row:
		return v.isUnsafe()
	case *Row:
		return v.isUnsafe()
	case Rows:
		return v.isUnsafe()
	case *Rows:
		return v.isUnsafe()
	case NamedStmt:
		return v.isUnsafe()

		//return v.Stmt.unsafe
	case *NamedStmt:
		return v.isUnsafe()
		// note(jae): 2022-10-22
		// Original SQLX checked for this:
		// return v.Stmt.unsafe
	case Stmt:
		return v.isUnsafe()
	case *Stmt:
		return v.isUnsafe()
	// todo(jae): 2022-10-22
	// implement qStmt support if needed
	/* case qStmt:
		return v.unsafe
	case *qStmt:
		return v.unsafe */
	case DB:
		return v.isUnsafe()
	case *DB:
		return v.isUnsafe()
	case Tx:
		return v.isUnsafe()
	case *Tx:
		return v.isUnsafe()
	case sql.Rows, *sql.Rows:
		return false
	default:
		return false
	}
}

// Rebind a query within a transaction's bindvar type.
func (tx *Tx) Rebind(query string) string {
	return Rebind(tx.bindType, query)
}

// Rebind a query from the default bindtype (QUESTION) to the target bindtype.
func Rebind(bindType int, query string) string {
	switch bindType {
	case QUESTION, UNKNOWN:
		return query
	}
	// note(jae): 2022-10-22
	// Borrowed from sqlx directly. We could probably write a parser
	// that's faster than this implementation later.

	// Add space enough for 10 params before we have to allocate
	rqb := make([]byte, 0, len(query)+10)

	var i, j int

	for i = strings.Index(query, "?"); i != -1; i = strings.Index(query, "?") {
		rqb = append(rqb, query[:i]...)

		switch bindType {
		case DOLLAR:
			rqb = append(rqb, '$')
		case NAMED:
			rqb = append(rqb, ':', 'a', 'r', 'g')
		case AT:
			rqb = append(rqb, '@', 'p')
		}

		j++
		rqb = strconv.AppendInt(rqb, int64(j), 10)

		query = query[i+1:]
	}

	return string(append(rqb, query...))
}

// colScanner is an interface used by MapScan and SliceScan
type colScanner interface {
	Columns() ([]string, error)
	Scan(dest ...interface{}) error
	Err() error
}

// SliceScan a row, returning a []interface{} with values similar to MapScan.
// This function is primarily intended for use where the number of columns
// is not known.  Because you can pass an []interface{} directly to Scan,
// it's recommended that you do that as it will not have to allocate new
// slices per row.
func SliceScan(r colScanner) ([]interface{}, error) {
	// ignore r.started, since we needn't use reflect for anything.
	columns, err := r.Columns()
	if err != nil {
		return []interface{}{}, err
	}

	ptrToValues := make([]interface{}, len(columns))
	values := make([]interface{}, len(columns))
	for i := 0; i < len(ptrToValues); i++ {
		// note(jae): 2022-10-29
		// Instead of new() we just create a backing interface{}
		// array, allocate all at once, and use that.
		// values[i] = new(interface{})
		ptrToValues[i] = &values[i]
	}

	err = r.Scan(ptrToValues...)
	if err != nil {
		return []interface{}{}, err
	}
	if err := r.Err(); err != nil {
		return []interface{}{}, err
	}

	// note(jae): 2022-10-29
	// This stops making interfaces pointers to the value
	// and just the value, we no longer need to do this.
	//for i := range columns {
	//	ptrToValues[i] = *(ptrToValues[i].(*interface{}))
	//}

	return values, nil
}
