package sqlsw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
	"github.com/silbinarywolf/sqlsw/internal/dbreflect"
	"github.com/silbinarywolf/sqlsw/internal/sqlparser"
)

var _scannerInterface = dbreflect.TypeOf((*sql.Scanner)(nil)).Elem()

type DB struct {
	db *sql.DB
	dataAndCaching
}

func newDB(dbDriver *sql.DB, driverName string) (*DB, error) {
	bindType, ok := getBindTypeFromDriverName(driverName)
	if !ok {
		return nil, errors.New("unable to get bind type for driver: " + driverName + "\nUse RegisterBindType to define how your database handles bound parameters.")
	}
	db := &DB{}
	db.db = dbDriver
	db.bindType = bindType
	db.reflector = dbreflect.NewReflectModule(dbreflect.Options{})
	return db, nil
}

// Open opens a database specified by its database driver name and a
// driver-specific data source name, usually consisting of at least a
// database name and connection information.
//
// The returned DB is safe for concurrent use by multiple goroutines
// and maintains its own pool of idle connections. Thus, the Open
// function should be called just once. It is rarely necessary to
// close a DB.
func Open(driverName, dataSourceName string) (*DB, error) {
	dbDriver, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	db, err := newDB(dbDriver, driverName)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) PingContext(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

// DB returns the underlying "database/sql" handle
//
// todo(jae): 2022-10-22
// Decide if we need to expose this.
// func (db *DB) DB() *sql.DB { return db.db }

// dataAndCaching is extra information stored on db and passed around statements and transactions
type dataAndCaching struct {
	// bindType is whether parameters are bound with ?, $, @, :Named, etc
	bindType bindtype.Kind
	// options contains things like "allowUnknownFields"
	options
	// caching holds any structs that cache data needed for Scan
	caching
}

type options struct {
	// allowUnknownFields allows skipping of fields not in struct
	allowUnknownFields bool
}

type optionsObject interface {
	getOptionsData() *options
}

func (opts *options) getOptionsData() *options {
	return opts
}

// caching is extra information stored on db and passed around statements and transactions
type caching struct {
	// reflector handles reflection logic and caching
	reflector *dbreflect.ReflectModule
}

func (c *caching) getCachingData() caching {
	return *c
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	rows *sql.Rows
	options
	// caching holds any structs that cache data needed for Scan
	caching
}

type cachingObject interface {
	getCachingData() caching
}

func newRows(rows *sql.Rows, optionsData options, cachingData caching) *Rows {
	r := &Rows{}
	r.rows = rows
	r.options = optionsData
	r.caching = cachingData
	return r
}

// Close closes the Rows, preventing further enumeration. If Next is called
// and returns false and there are no further result sets,
// the Rows are closed automatically and it will suffice to check the
// result of Err. Close is idempotent and does not affect the result of Err.
func (rows *Rows) Close() error {
	return rows.rows.Close()
}

// Err returns the error, if any, that was encountered during iteration.
// Err may be called after an explicit or implicit Close.
func (rows *Rows) Err() error {
	return rows.rows.Err()
}

// Next prepares the next result row for reading with the Scan method. It
// returns true on success, or false if there is no next result row or an error
// happened while preparing it. Err should be consulted to distinguish between
// the two cases.
//
// Every call to Scan, even the first one, must be preceded by a call to Next.
func (rows *Rows) Next() bool {
	return rows.rows.Next()
}

// Tx is an in-progress database transaction.
type Tx struct {
	underlying *sql.Tx
	dataAndCaching
}

func newTx(tx *sql.Tx, caching dataAndCaching) *Tx {
	t := &Tx{}
	t.underlying = tx
	t.dataAndCaching = caching
	return t
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	return tx.underlying.Commit()
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback() error {
	return tx.underlying.Rollback()
}

// NamedStmt is a prepared statement.
// A NamedStmt is safe for concurrent use by multiple goroutines.
type NamedStmt struct {
	underlying *sql.Stmt
	parameters []string
	options
	// caching holds any structs that cache data needed for Scan
	caching
}

func newNamedStmt(stmt *sql.Stmt, parameters []string, optionsData options, cachingData caching) *NamedStmt {
	s := &NamedStmt{}
	s.underlying = stmt
	s.parameters = parameters
	s.options = optionsData
	s.caching = cachingData
	return s
}

// Stmt gets the underlying "database/sql" statement
func (stmt *NamedStmt) Stmt() *sql.Stmt {
	return stmt.underlying
}

// Close closes the statement.
func (stmt *NamedStmt) Close() error {
	return stmt.underlying.Close()
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	return newTx(tx, db.dataAndCaching), nil
}

// NamedPrepareContext creates a prepared statement for later queries or executions.
func (db *DB) NamedPrepareContext(ctx context.Context, query string) (*NamedStmt, error) {
	parseResult, err := parseNamedQuery(query, sqlparser.Options{
		BindType: db.bindType,
	})
	if err != nil {
		return nil, err
	}
	stmt, err := db.db.PrepareContext(ctx, parseResult.Query())
	if err != nil {
		return nil, err
	}
	return newNamedStmt(stmt, parseResult.Parameters(), db.options, db.caching), nil
}

// NamedPrepareContext creates a prepared statement for later queries or executions.
func (tx *Tx) NamedPrepareContext(ctx context.Context, query string) (*NamedStmt, error) {
	parseResult, err := parseNamedQuery(query, sqlparser.Options{
		BindType: tx.bindType,
	})
	if err != nil {
		return nil, err
	}
	stmt, err := tx.underlying.PrepareContext(ctx, parseResult.Query())
	if err != nil {
		return nil, err
	}
	return newNamedStmt(stmt, parseResult.Parameters(), tx.options, tx.caching), nil
}

// NamedExecContext executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func (db *DB) NamedExecContext(ctx context.Context, query string, structOrMapOrSlice interface{}) (sql.Result, error) {
	query, argList, err := transformNamedQueryAndParams(db.reflector, db.bindType, query, structOrMapOrSlice)
	if err != nil {
		return nil, err
	}
	sqlResult, err := db.db.ExecContext(ctx, query, argList...)
	if err != nil {
		return nil, err
	}
	return sqlResult, nil
}

// NamedQueryContext executes a query that returns rows, typically a SELECT.
func (db *DB) NamedQueryContext(ctx context.Context, query string, structOrMapOrSlice interface{}) (*Rows, error) {
	query, argList, err := transformNamedQueryAndParams(db.reflector, db.bindType, query, structOrMapOrSlice)
	if err != nil {
		return nil, err
	}
	rows, err := db.db.QueryContext(ctx, query, argList...)
	if err != nil {
		return nil, err
	}
	return newRows(rows, db.options, db.caching), nil
}

// NamedQueryContext executes a query that returns rows, typically a SELECT.
func (tx *Tx) NamedQueryContext(ctx context.Context, query string, structOrMapOrSlice interface{}) (*Rows, error) {
	query, argList, err := transformNamedQueryAndParams(tx.reflector, tx.bindType, query, structOrMapOrSlice)
	if err != nil {
		return nil, err
	}
	rows, err := tx.underlying.QueryContext(ctx, query, argList...)
	if err != nil {
		return nil, err
	}
	return newRows(rows, tx.options, tx.caching), nil
}

// NamedQueryContext executes a query that returns rows, typically a SELECT.
func (stmt *NamedStmt) NamedQueryContext(ctx context.Context, structOrMapOrSlice interface{}) (*Rows, error) {
	argList, err := getArgumentListFromParameters(stmt.reflector, stmt.parameters, structOrMapOrSlice)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.underlying.QueryContext(ctx, argList...)
	if err != nil {
		return nil, err
	}
	return newRows(rows, stmt.options, stmt.caching), nil
}

// NamedExecContext executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func (stmt *NamedStmt) NamedExecContext(ctx context.Context, structOrMapOrSlice interface{}) (sql.Result, error) {
	argList, err := getArgumentListFromParameters(stmt.reflector, stmt.parameters, structOrMapOrSlice)
	if err != nil {
		return nil, err
	}
	sqlResult, err := stmt.underlying.ExecContext(ctx, argList...)
	if err != nil {
		return nil, err
	}
	return sqlResult, nil
}

// namedQuery is an interface for calling NamedQueryContext with a database or transaction
type namedQuery interface {
	NamedQueryContext(ctx context.Context, query string, structOrMapOrSlice interface{}) (*Rows, error)
}

// NamedQueryContext executes a query that returns rows, typically a SELECT.
func NamedQueryContext(ctx context.Context, dbOrTx namedQuery, query string, structOrMapOrSlice interface{}) (*Rows, error) {
	switch dbOrTx.(type) {
	case *DB:
		return dbOrTx.NamedQueryContext(ctx, query, structOrMapOrSlice)
	case *Tx:
		return dbOrTx.NamedQueryContext(ctx, query, structOrMapOrSlice)
	}
	return nil, errors.New("unable to execute NamedQueryContext, must be sqlsw database or transaction")
}

// Row is the result of calling QueryRow to select a single row.
type Row struct {
	err  error
	rows Rows
}

// getOptionsData is so a Row can be passed to SQLX compat layer
func (row *Row) getOptionsData() *options {
	return &row.rows.options
}

// ScanStruct copies the columns in the current row into the given struct.
//
// If more than one row matches the query,
// Scan uses the first row and discards the rest. If no row matches
// the query, Scan returns ErrNoRows.
func (r *Row) ScanStruct(dest interface{}) error {
	if r.err != nil {
		return r.err
	}
	defer r.rows.Close()
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	err := r.rows.ScanStruct(dest)
	if err != nil {
		return err
	}
	// Make sure the query can be processed to completion with no errors.
	return r.rows.Close()
}

// Err provides a way for wrapping packages to check for
// query errors without calling Scan.
// Err returns the error, if any, that was encountered while running the query.
// If this error is not nil, this error will also be returned from Scan.
func (r *Row) Err() error {
	return r.err
}

// NamedQueryRowContext executes a named prepared query statement with the given arguments.
func (db *DB) NamedQueryRowContext(ctx context.Context, query string, structOrMapOrSlice interface{}) *Row {
	rows, err := db.NamedQueryContext(ctx, query, structOrMapOrSlice)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{rows: *rows}
}

// NamedQueryRowContext executes a named prepared query statement with the given arguments.
func (tx *Tx) NamedQueryRowContext(ctx context.Context, query string, structOrMapOrSlice interface{}) *Row {
	rows, err := tx.NamedQueryContext(ctx, query, structOrMapOrSlice)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{rows: *rows}
}

// NamedQueryRowContext executes a named prepared query statement with the given arguments.
func (stmt *NamedStmt) NamedQueryRowContext(ctx context.Context, structOrMapOrSlice interface{}) *Row {
	rows, err := stmt.NamedQueryContext(ctx, structOrMapOrSlice)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{rows: *rows}
}

// ScanSlice copies the columns in the current row into the given struct.
func (rows *Rows) ScanSlice(ptrToSlice interface{}) error {
	refType := dbreflect.TypeOf(ptrToSlice)
	if refType.Kind() != reflect.Ptr {
		return errors.New("ScanSlice: must pass a pointer, not a value")
	}
	refType = refType.Elem()
	if kind := refType.Kind(); kind != reflect.Slice {
		return errors.New("ScanSlice: must pass a pointer to slice, not pointer to " + kind.String())
	}
	isPtr := false
	sliceElem := refType.Elem()
	if sliceElem.Kind() == reflect.Ptr {
		sliceElem = sliceElem.Elem()
		isPtr = true
	}
	direct := reflect.Indirect(reflect.ValueOf(ptrToSlice))
	direct.SetLen(0)

	// note(jae): 2022-10-23
	// isScannable checks if a type implements `Scan(src interface{}) error` and
	// if it's not a struct
	isScannable := dbreflect.PtrTo(sliceElem).Implements(_scannerInterface) ||
		sliceElem.Kind() != reflect.Struct
	if !isScannable {
		columnNames, err := rows.rows.Columns()
		if err != nil {
			return err
		}
		structData, err := rows.reflector.GetStruct(sliceElem)
		if err != nil {
			return err
		}
		var (
			values []interface{}
			// temporary array used on stack
			valuesUnderlying [16]interface{}
			// skippedFieldValue is used to hold skipped values
			skippedFieldValue interface{}
		)
		if len(columnNames) >= len(valuesUnderlying) {
			values = valuesUnderlying[:len(columnNames)]
		} else {
			values = make([]interface{}, len(columnNames))
		}
		for rows.Next() {
			vp := dbreflect.New(sliceElem)
			// Fill struct with values
			{
				for i, columnName := range columnNames {
					field, ok := structData.GetFieldByName(columnName)
					if !ok {
						if rows.allowUnknownFields {
							values[i] = &skippedFieldValue
							continue
						}
						return fmt.Errorf(`missing column name "%s" in %T`, columnName, ptrToSlice)
					}
					values[i] = field.Addr(vp)
				}
				if err := rows.rows.Scan(values...); err != nil {
					return err
				}
			}
			if isPtr {
				direct.Set(reflect.Append(direct, vp.UnderlyingValue()))
			} else {
				v := dbreflect.Indirect(vp)
				direct.Set(reflect.Append(direct, v.UnderlyingValue()))
			}
		}
	} else {
		// note(jae): 2022-10-23
		// Handles cases such as:
		// - []int32, []int64, time.Time
		for rows.Next() {
			vp := dbreflect.New(sliceElem)
			if err := rows.rows.Scan(vp.Interface()); err != nil {
				return err
			}
			if isPtr {
				direct.Set(reflect.Append(direct, vp.UnderlyingValue()))
			} else {
				direct.Set(reflect.Append(direct, reflect.Indirect(vp.UnderlyingValue())))
			}
		}
	}
	return rows.Err()
}

// ScanStruct copies the columns in the current row into the given struct.
func (rows *Rows) ScanStruct(ptrValue interface{}) error {
	refType := dbreflect.TypeOf(ptrValue)
	if refType.Kind() != reflect.Ptr {
		return errors.New("ScanStruct: must pass a pointer, not a value")
	}
	refType = refType.Elem()
	if refType.Kind() != reflect.Struct {
		return errors.New("ScanStruct: must pass a pointer to struct, not " + refType.Kind().String())
	}
	columnNames, err := rows.rows.Columns()
	if err != nil {
		return err
	}
	// Get values
	var (
		values []interface{}
		// temporary array used on stack
		valuesUnderlying [16]interface{}
		// skippedFieldValue is used to hold skipped values
		skippedFieldValue interface{}
	)
	{
		if len(columnNames) >= len(valuesUnderlying) {
			values = valuesUnderlying[:len(columnNames)]
		} else {
			values = make([]interface{}, len(columnNames))
		}
		structData, err := rows.reflector.GetStruct(refType)
		if err != nil {
			return err
		}
		reflectArgs := dbreflect.ValueOf(ptrValue)
		for i, columnName := range columnNames {
			field, ok := structData.GetFieldByName(columnName)
			if !ok {
				if rows.allowUnknownFields {
					values[i] = &skippedFieldValue
					continue
				}
				return fmt.Errorf(`missing column name "%s" in %T`, columnName, ptrValue)
			}
			values[i] = field.Addr(reflectArgs)
		}
	}
	err = rows.rows.Scan(values...)
	if err != nil {
		return err
	}
	return rows.Err()
}

type unexpectedNamedParameterError struct{}

func (err *unexpectedNamedParameterError) Error() string {
	return `unexpected named parameter, expected map, array, slice, struct or pointer to struct`
}

type missingValueError struct {
	fieldName string
}

func (err *missingValueError) Error() string {
	return `missing value for named parameter: ` + err.fieldName
}

func parseNamedQuery(query string, options sqlparser.Options) (sqlparser.ParseResult, error) {
	// not-cached
	//
	//
	return sqlparser.Parse(query, options)

	// cached
	// - IMO, the savings aren't high enough to justify this
	// - this logic is also incorrect, doesn't use bind type in key
	// - requires re-adding "var cachedNamedQuery sync.Map"
	//
	// BenchmarkNamedQueryContextWithScanStruct-12    	    1014	   1169991 ns/op	     728 B/op	      21 allocs/op
	//
	/* unassertedParsedResult, ok := cachedNamedQuery.Load(query)
	if ok {
		return unassertedParsedResult.(sqlparser.ParseResult), nil
	}
	parseResult, err := sqlparser.Parse(query, options)
	if err != nil {
		return parseResult, err
	}
	cachedNamedQuery.Store(query, parseResult)
	return parseResult, nil */
}

func getArgumentListFromParameters(reflector *dbreflect.ReflectModule, parameterNames []string, mapOrStructOrSlice interface{}) ([]interface{}, error) {
	var argList []interface{}
	switch mapOrStructOrSlice := mapOrStructOrSlice.(type) {
	case map[string]interface{}:
		// Fast-path for map[string]interface{}
		argList = make([]interface{}, 0, len(parameterNames))
		for _, fieldName := range parameterNames {
			v, ok := mapOrStructOrSlice[fieldName]
			if !ok {
				return nil, &missingValueError{fieldName: fieldName}
			}
			argList = append(argList, v)
		}
	case map[string]string:
		// Fast-path for map[string]string
		argList = make([]interface{}, 0, len(parameterNames))
		for _, fieldName := range parameterNames {
			v, ok := mapOrStructOrSlice[fieldName]
			if !ok {
				return nil, &missingValueError{fieldName: fieldName}
			}
			argList = append(argList, v)
		}
	default:
		t := reflect.TypeOf(mapOrStructOrSlice)
		k := t.Kind()
		switch k {
		case reflect.Map:
			if mapKeyKind := t.Key().Kind(); mapKeyKind != reflect.String {
				return nil, errors.New(`unsupported map key type "` + mapKeyKind.String() + `", must be "string", ie. map[string]interface{} or map[string]string`)
			}
			// note(jae): 2022-10-15
			// Slow-path that SQLx uses on map types.
			// Benchmarking shows this style takes ~100x longer
			//
			// Type Assert:
			// - 1000000000	         0.3219 ns/op	       0 B/op	       0 allocs/op
			//
			// ValueOf.Convert:
			// - 44560135	         26.44 ns/op	       0 B/op	       0 allocs/op
			mtype := reflect.TypeOf(map[string]interface{}{})
			if !reflect.TypeOf(mapOrStructOrSlice).ConvertibleTo(mtype) {
				return nil, errors.New(`invalid map given, unable to convert to map[string]interface{}`)
			}
			argMap := reflect.ValueOf(mapOrStructOrSlice).Convert(mtype).Interface().(map[string]interface{})
			for _, fieldName := range parameterNames {
				v, ok := argMap[fieldName]
				if !ok {
					return nil, &missingValueError{fieldName: fieldName}
				}
				argList = append(argList, v)
			}
		case reflect.Array, reflect.Slice:
			arrayValue := reflect.ValueOf(mapOrStructOrSlice)
			arrayLen := arrayValue.Len()
			if arrayLen == 0 {
				return nil, fmt.Errorf("length of array is 0: %#v", mapOrStructOrSlice)
			}
			// note(jae): 2022-10-15
			// This won't be an exact fit for arguments and will over-allocate
			// if the same parameter is used twice.
			argList = make([]interface{}, 0, len(parameterNames)*arrayLen)
			for i := 0; i < arrayLen; i++ {
				switch args := mapOrStructOrSlice.(type) {
				case map[string]interface{}:
					for _, fieldName := range parameterNames {
						v, ok := args[fieldName]
						if !ok {
							return nil, &missingValueError{fieldName: fieldName}
						}
						argList = append(argList, v)
					}
				case map[string]string:
					for _, fieldName := range parameterNames {
						v, ok := args[fieldName]
						if !ok {
							return nil, &missingValueError{fieldName: fieldName}
						}
						argList = append(argList, v)
					}
				default:
					// note(jae): 2022-10-15
					// Slow-path that SQLx uses on map types
					//
					// Benchmarking shows this style takes ~100x longer
					//
					// Type Assert:
					// - 1000000000	         0.3219 ns/op	       0 B/op	       0 allocs/op
					//
					// ValueOf.Convert:
					// - 44560135	         26.44 ns/op	       0 B/op	       0 allocs/op
					if mtype := reflect.TypeOf(map[string]interface{}{}); reflect.TypeOf(args).ConvertibleTo(mtype) {
						argMap := reflect.ValueOf(args).Convert(mtype).Interface().(map[string]interface{})
						for _, fieldName := range parameterNames {
							v, ok := argMap[fieldName]
							if !ok {
								return nil, &missingValueError{fieldName: fieldName}
							}
							argList = append(argList, v)
						}
					} else {
						panic("TODO: Bind struct variables")
					}
				}
			}
		default:
			if k != reflect.Ptr && k != reflect.Struct {
				return nil, &unexpectedNamedParameterError{}
			}
			if k == reflect.Ptr {
				t = t.Elem()
				if t.Kind() == reflect.Ptr {
					// Disallow nested pointers
					//
					// - MyStruct, *MyStruct = allowed
					// - **MyStruct, ***MyStruct = not allowed
					return nil, &unexpectedNamedParameterError{}
				}
			}
			if t.Kind() != reflect.Struct {
				return nil, &unexpectedNamedParameterError{}
			}
			structData, err := reflector.GetStruct(dbreflect.TypeOf(mapOrStructOrSlice))
			if err != nil {
				return nil, err
			}
			reflectArgs := dbreflect.ValueOf(mapOrStructOrSlice)
			// note(jae): 2022-10-15
			// This won't be an exact fit for arguments and will over-allocate
			// if the same parameter is used twice.
			argList = make([]interface{}, 0, len(parameterNames))
			for _, parameterName := range parameterNames {
				field, ok := structData.GetFieldByName(parameterName)
				if !ok {
					return nil, errors.New(parameterName + " was not found on struct")
				}
				argList = append(argList, field.Interface(reflectArgs))
			}
		}
	}
	return argList, nil
}

func transformNamedQueryAndParams(reflector *dbreflect.ReflectModule, bindType bindtype.Kind, query string, mapOrStructOrSlice interface{}) (string, []interface{}, error) {
	parseResult, err := parseNamedQuery(query, sqlparser.Options{
		BindType: bindType,
	})
	if err != nil {
		return "", nil, err
	}
	argList, err := getArgumentListFromParameters(reflector, parseResult.Parameters(), mapOrStructOrSlice)
	if err != nil {
		return "", nil, err
	}
	return parseResult.Query(), argList, err
}

// testOrBench is an interface for testing.T or testing.B
type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// TestOnlyResetCache will reset all caching logic on the DB struct
//
// This should be used for testing and benchmarking purposes only.
func TestOnlyResetCache(t testOrBench, db *DB) {
	db.reflector.ResetCache()
}
