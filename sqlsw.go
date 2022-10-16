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

type DB struct {
	*sql.DB
	dbData
}

type dbData struct {
	// bindType is whether parameters are bound with ?, $, @, :Named, etc
	bindType bindtype.Kind
	// reflector handles reflection logic and caching
	reflector *dbreflect.ReflectModule
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	*sql.Rows
	dbData
}

func Open(driverName, dataSourceName string) (*DB, error) {
	dbDriver, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	bindType, ok := getBindTypeFromDriverName(driverName)
	if !ok {
		return nil, errors.New("unable to get bind type for driver: " + driverName + "\nUse RegisterBindType to define how your database handles bound parameters.")
	}
	db := &DB{}
	db.DB = dbDriver
	db.bindType = bindType
	db.reflector = &dbreflect.ReflectModule{}
	return db, nil
}

func (db *DB) NamedQueryContext(ctx context.Context, query string, args interface{}) (*Rows, error) {
	query, argList, err := transformNamedQueryAndParams(db.reflector, db.bindType, query, args)
	if err != nil {
		return nil, err
	}
	sqlRows, err := db.DB.QueryContext(ctx, query, argList...)
	if err != nil {
		return nil, err
	}
	r := &Rows{}
	r.Rows = sqlRows
	r.dbData = db.dbData
	return r, nil
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
	columnNames, err := rows.Columns()
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
				values[i] = &skippedFieldValue
				continue
			}
			values[i] = field.Addr(reflectArgs)
		}
	}
	err = rows.Scan(values...)
	if err != nil {
		return err
	}
	return rows.Err()
}

type unexpectedNamedParameterError struct {
}

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

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// TestOnlyResetCache will reset all caching logic on the DB struct
//
// This should be used for testing and benchmarking purposes only.
func TestOnlyResetCache(t testOrBench, db *DB) {
	db.reflector = &dbreflect.ReflectModule{}
}

func transformNamedQueryAndParams(reflector *dbreflect.ReflectModule, bindType bindtype.Kind, query string, args interface{}) (string, []interface{}, error) {
	parseResult, err := parseNamedQuery(query, sqlparser.Options{
		BindType: bindType,
	})
	if err != nil {
		return "", nil, err
	}
	parameterNames := parseResult.Parameters()

	t := reflect.TypeOf(args)
	k := t.Kind()
	var argList []interface{}
	switch k {
	case reflect.Map:
		if mapKeyKind := t.Key().Kind(); mapKeyKind != reflect.String {
			return "", nil, errors.New(`unsupported map key type "` + mapKeyKind.String() + `", must be "string", ie. map[string]interface{}`)
		}
		// note(jae): 2022-10-15
		// This won't be an exact fit for arguments and will over-allocate
		// if the same parameter is used twice.
		argList = make([]interface{}, 0, len(parameterNames))
		switch args := args.(type) {
		case map[string]interface{}:
			for _, fieldName := range parameterNames {
				v, ok := args[fieldName]
				if !ok {
					return "", nil, &missingValueError{fieldName: fieldName}
				}
				argList = append(argList, v)
			}
		case map[string]string:
			for _, fieldName := range parameterNames {
				v, ok := args[fieldName]
				if !ok {
					return "", nil, &missingValueError{fieldName: fieldName}
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
			mtype := reflect.TypeOf(map[string]interface{}{})
			if !reflect.TypeOf(args).ConvertibleTo(mtype) {
				return "", nil, errors.New(`invalid map given, unable to convert to map[string]interface{}`)
			}
			argMap := reflect.ValueOf(args).Convert(mtype).Interface().(map[string]interface{})
			for _, fieldName := range parameterNames {
				v, ok := argMap[fieldName]
				if !ok {
					return "", nil, &missingValueError{fieldName: fieldName}
				}
				argList = append(argList, v)
			}
		}
	case reflect.Array, reflect.Slice:
		arrayValue := reflect.ValueOf(args)
		arrayLen := arrayValue.Len()
		if arrayLen == 0 {
			return "", nil, fmt.Errorf("length of array is 0: %#v", args)
		}
		// note(jae): 2022-10-15
		// This won't be an exact fit for arguments and will over-allocate
		// if the same parameter is used twice.
		argList = make([]interface{}, 0, len(parameterNames)*arrayLen)
		for i := 0; i < arrayLen; i++ {
			switch args := args.(type) {
			case map[string]interface{}:
				for _, fieldName := range parameterNames {
					v, ok := args[fieldName]
					if !ok {
						return "", nil, &missingValueError{fieldName: fieldName}
					}
					argList = append(argList, v)
				}
			case map[string]string:
				for _, fieldName := range parameterNames {
					v, ok := args[fieldName]
					if !ok {
						return "", nil, &missingValueError{fieldName: fieldName}
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
							return "", nil, &missingValueError{fieldName: fieldName}
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
			return "", nil, &unexpectedNamedParameterError{}
		}
		if k == reflect.Ptr {
			t = t.Elem()
			if t.Kind() == reflect.Ptr {
				// Disallow nested pointers
				//
				// - MyStruct, *MyStruct = allowed
				// - **MyStruct, ***MyStruct = not allowed
				return "", nil, &unexpectedNamedParameterError{}
			}
		}
		if t.Kind() != reflect.Struct {
			return "", nil, &unexpectedNamedParameterError{}
		}
		structData, err := reflector.GetStruct(dbreflect.TypeOf(args))
		if err != nil {
			return "", nil, err
		}
		reflectArgs := dbreflect.ValueOf(args)
		// note(jae): 2022-10-15
		// This won't be an exact fit for arguments and will over-allocate
		// if the same parameter is used twice.
		argList = make([]interface{}, 0, len(parameterNames))
		for _, parameterName := range parameterNames {
			field, ok := structData.GetFieldByName(parameterName)
			if !ok {
				return "", nil, errors.New(parameterName + " was not found on struct")
			}
			argList = append(argList, field.Interface(reflectArgs))
		}
	}
	return parseResult.Query(), argList, nil
}
