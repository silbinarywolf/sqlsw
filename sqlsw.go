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
	// handle is a database handle from database/sql
	db *sql.DB
	dbData
}

type dbData struct {
	// bindType is whether parameters are bound with ?, $, @, :Named, etc
	bindType bindtype.Kind
	// reflecter handles reflection logic and caching
	reflecter *dbreflect.ReflectModule
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows struct {
	rows
	dbData
}

// rows exists to add another layer of indirection so a user can't change
// the pointer to Rows it's holding
type rows struct {
	*sql.Rows
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
	db.db = dbDriver
	db.bindType = bindType
	db.reflecter = &dbreflect.ReflectModule{}
	return db, nil
}

func (db *DB) NamedQueryContext(ctx context.Context, query string, args interface{}) (*Rows, error) {
	query, argList, err := transformNamedQueryAndParams(db.bindType, query, args)
	if err != nil {
		return nil, err
	}
	sqlRows, err := db.db.QueryContext(ctx, query, argList...)
	if err != nil {
		return nil, err
	}
	r := &Rows{}
	r.rows.Rows = sqlRows
	r.dbData = db.dbData
	return r, nil
}

func (rows *Rows) ScanStruct(args interface{}) error {
	columnNames, err := rows.rows.Columns()
	if err != nil {
		return err
	}
	var (
		values           []interface{}
		valuesUnderlying [8]interface{}
	)
	if len(columnNames) >= len(valuesUnderlying) {
		values = valuesUnderlying[:len(columnNames)]
	} else {
		values = make([]interface{}, len(columnNames))
	}
	err = rows.Scan(values...)
	if err != nil {
		return err
	}
	panic("todo: put values in struct")
	return rows.Err()
	/* reflectArgs := dbreflect.ValueOf(args)
	argList = make([]interface{}, 0, len(parameterNames))
	for _, parameterName := range rows.argList {
		field, ok := structData.GetFieldByTagName(parameterName)
		if !ok {
			return errors.New(parameterName + " was not found on struct")
		}
		argList = append(argList, field.Interface(reflectArgs))
	} */
	/*
		v := reflect.ValueOf(dest)

		if v.Kind() != reflect.Ptr {
			return errors.New("must pass a pointer, not a value, to StructScan destination")
		}

		v = v.Elem()

		if !r.started {
			columns, err := r.Columns()
			if err != nil {
				return err
			}
			m := r.Mapper

			r.fields = m.TraversalsByName(v.Type(), columns)
			// if we are not unsafe and are missing fields, return an error
			if f, err := missingFields(r.fields); err != nil && !r.unsafe {
				return fmt.Errorf("missing destination name %s in %T", columns[f], dest)
			}
			r.values = make([]interface{}, len(columns))
			r.started = true
		}

		err := fieldsByTraversal(v, r.fields, r.values, true)
		if err != nil {
			return err
		}
		// scan into the struct field pointers and append to our results
		err = r.Scan(r.values...)
		if err != nil {
			return err
		}
		return r.Err()
	*/
	return nil
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

func transformNamedQueryAndParams(reflecter *dbreflect.ReflectModule, bindType bindtype.Kind, query string, args interface{}) (string, []interface{}, error) {
	parseResult, err := sqlparser.Parse(query, sqlparser.Options{
		BindType: bindType,
	})
	if err != nil {
		return "", nil, err
	}
	transformedQuery := parseResult.Query
	parameterNames := parseResult.Parameters
	// note(jae): 2022-10-15
	// This won't be an exact fit for arguments and will over-allocate
	// if the same parameter is used twice.
	argList := make([]interface{}, 0, len(parameterNames))
	t := reflect.TypeOf(args)
	k := t.Kind()
	switch k {
	case reflect.Map:
		if mapKeyKind := t.Key().Kind(); mapKeyKind != reflect.String {
			return "", nil, errors.New(`unsupported map key type "` + mapKeyKind.String() + `", must be "string", ie. map[string]interface{}`)
		}
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
		argList := make([]interface{}, 0, len(parameterNames)*arrayLen)
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
					return "", nil, errors.New(`invalid map given, unable to convert to map[string]interface{}`)
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
		structData, err := reflecter.GetStruct(dbreflect.TypeOf(args))
		if err != nil {
			return "", nil, err
		}
		reflectArgs := dbreflect.ValueOf(args)
		argList = make([]interface{}, 0, len(parameterNames))
		for _, parameterName := range parameterNames {
			field, ok := structData.GetFieldByTagName(parameterName)
			if !ok {
				return "", nil, errors.New(parameterName + " was not found on struct")
			}
			argList = append(argList, field.Interface(reflectArgs))
		}
	}
	return transformedQuery, argList, nil
}
