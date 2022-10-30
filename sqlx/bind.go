package sqlx

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Bindvar types supported by Rebind, BindMap and BindStruct.
const (
	UNKNOWN  = 0
	QUESTION = 1
	DOLLAR   = 2
	NAMED    = 3
	AT       = 4
)

// Rebind a query from the default bindtype (QUESTION) to the target bindtype.
func Rebind(bindType int, query string) string {
	switch bindType {
	case QUESTION, UNKNOWN:
		return query
	}

	var (
		stackQuery [128]byte
		// varCount is how many ? parameters are in the query string
		varCount = 0
	)

	// Setup query replacement buffer
	var queryReplace []byte
	if len(query) >= len(stackQuery) {
		// If we're likely going to go over stack bytes
		// just allocate once here
		//
		// note from SQLX: Add space enough for 10 params before we have to allocate
		queryReplace = make([]byte, 0, len(query)+10)
	} else {
		// Use bytes on the stack while building the new query string
		queryReplace = stackQuery[:0]
	}

	for pos := 0; pos < len(query); {
		r, size := utf8.DecodeRuneInString(query[pos:])
		pos += size

		// todo(jae): 2022-10-30
		// Once sqlparser has handled escaped strings correctly, we should avoid
		// handling ? within strings here too.
		switch r {
		case '?':
			switch bindType {
			case DOLLAR:
				queryReplace = append(queryReplace, '$')
			case NAMED:
				queryReplace = append(queryReplace, ':', 'a', 'r', 'g')
			case AT:
				queryReplace = append(queryReplace, '@', 'p')
			}
			varCount++
			queryReplace = strconv.AppendInt(queryReplace, int64(varCount), 10)
			continue
		}
		queryReplace = utf8.AppendRune(queryReplace, r)
	}

	return string(queryReplace)
}

/* func appendString(slice []byte, str string) []byte {
	r := slice
	for i := range str {
		r = append(r, str[i])
	}
	return r
} */

// rebindOriginal a query from the default bindtype (QUESTION) to the target bindtype.
// From v1.3.5 of SQLX
func rebindOriginal(bindType int, query string) string {
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

// In expands slice values in args, returning the modified query string
// and a new arg list that can be executed by a database. The `query` should
// use the `?` bindVar.  The return value uses the `?` bindVar.
func In(query string, args ...interface{}) (string, []interface{}, error) {
	// argMeta stores reflect.Value and length for slices and
	// the value itself for non-slice arguments
	type argMeta struct {
		v      reflect.Value
		i      interface{}
		length int
	}

	var flatArgsCount int
	var anySlices bool

	var stackMeta [32]argMeta

	var meta []argMeta
	if len(args) <= len(stackMeta) {
		meta = stackMeta[:len(args)]
	} else {
		meta = make([]argMeta, len(args))
	}

	for i, arg := range args {
		if a, ok := arg.(driver.Valuer); ok {
			var err error
			arg, err = a.Value()
			if err != nil {
				return "", nil, err
			}
		}

		if v, ok := asSliceForIn(arg); ok {
			meta[i].length = v.Len()
			meta[i].v = v

			anySlices = true
			flatArgsCount += meta[i].length

			if meta[i].length == 0 {
				return "", nil, errors.New("empty slice passed to 'in' query")
			}
		} else {
			meta[i].i = arg
			flatArgsCount++
		}
	}

	// don't do any parsing if there aren't any slices;  note that this means
	// some errors that we might have caught below will not be returned.
	if !anySlices {
		return query, args, nil
	}

	newArgs := make([]interface{}, 0, flatArgsCount)

	var buf strings.Builder
	buf.Grow(len(query) + len(", ?")*flatArgsCount)

	var arg, offset int

	for i := strings.IndexByte(query[offset:], '?'); i != -1; i = strings.IndexByte(query[offset:], '?') {
		if arg >= len(meta) {
			// if an argument wasn't passed, lets return an error;  this is
			// not actually how database/sql Exec/Query works, but since we are
			// creating an argument list programmatically, we want to be able
			// to catch these programmer errors earlier.
			return "", nil, errors.New("number of bindVars exceeds arguments")
		}

		argMeta := meta[arg]
		arg++

		// not a slice, continue.
		// our questionmark will either be written before the next expansion
		// of a slice or after the loop when writing the rest of the query
		if argMeta.length == 0 {
			offset = offset + i + 1
			newArgs = append(newArgs, argMeta.i)
			continue
		}

		// write everything up to and including our ? character
		buf.WriteString(query[:offset+i+1])

		for si := 1; si < argMeta.length; si++ {
			buf.WriteString(", ?")
		}

		newArgs = appendReflectSlice(newArgs, argMeta.v, argMeta.length)

		// slice the query and reset the offset. this avoids some bookkeeping for
		// the write after the loop
		query = query[offset+i+1:]
		offset = 0
	}

	buf.WriteString(query)

	if arg < len(meta) {
		return "", nil, errors.New("number of bindVars less than number arguments")
	}

	return buf.String(), newArgs, nil
}

func asSliceForIn(i interface{}) (v reflect.Value, ok bool) {
	if i == nil {
		return reflect.Value{}, false
	}

	v = reflect.ValueOf(i)

	// note(jae): 2022-10-30
	// original SQLX code called "reflectx.Deref"
	// this is the equivalent logic
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only expand slices
	if t.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}

	// []byte is a driver.Value type so it should not be expanded
	if t == reflect.TypeOf([]byte{}) {
		return reflect.Value{}, false

	}

	return v, true
}

func appendReflectSlice(args []interface{}, v reflect.Value, vlen int) []interface{} {
	switch val := v.Interface().(type) {
	case []interface{}:
		args = append(args, val...)
	case []int:
		for i := range val {
			args = append(args, val[i])
		}
	case []string:
		for i := range val {
			args = append(args, val[i])
		}
	default:
		for si := 0; si < vlen; si++ {
			args = append(args, v.Index(si).Interface())
		}
	}

	return args
}
