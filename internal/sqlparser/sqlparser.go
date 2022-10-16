package sqlparser

import (
	"errors"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
)

// Test inlining/heap escapes
// go build -gcflags=-m=2

type Options struct {
	BindType bindtype.Kind

	// todo(jae): 2022-10-15
	// consider supporting legacy sqlx code
	sqlXBackwardsCompat bool
}

type ParseResult struct {
	query      string
	parameters []string
}

func (pr *ParseResult) Query() string {
	return pr.query
}

func (pr *ParseResult) Parameters() []string {
	return pr.parameters
}

// Parse will take a query string and return the query but replace the interpolated
// names with the bind type ($1, ?, @1, etc) and a list of parameters
func Parse(query string, options Options) (ParseResult, error) {
	// Store some data temporarily on the stack to reduce allocation
	// of bytes + overall count
	//
	// These get copied onto the heap later
	var (
		stackQuery      [256]byte
		stackParameters [16]string
	)
	// Setup query replacement buffer
	var queryReplace []byte
	if len(query) >= len(stackQuery) {
		// If we're likely going to go over stack bytes
		// just allocate once here
		queryReplace = make([]byte, 0, len(query))
	} else {
		// Use bytes on the stack while building the new query string
		queryReplace = stackQuery[:0]
	}
	// currentParamIndex is used for bind types that require positional
	// knowledge such as $ and @.
	// ie. $0
	currentParamIndex := 1
	bindType := options.BindType
	parameters := stackParameters[:0]
	for pos := 0; pos < len(query); {
		r, size := utf8.DecodeRuneInString(query[pos:])
		switch r {
		case ':':
			pos += size
			startPos := pos
			for pos < len(query) {
				r, size := utf8.DecodeRuneInString(query[pos:])
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					break
				}
				pos += size
			}
			if startPos == pos {
				r, size := utf8.DecodeRuneInString(query[pos:])
				if r == ':' {
					// Ignore :: case, ie.
					// - select 'SRID=4269;POINT(-123 34)'::geography from "MyTable"
					queryReplace = appendRune(queryReplace, r)
					queryReplace = appendRune(queryReplace, r)
					pos += size
					break
				}
				// Ignore non-letter and non-digit after :
				// - select : "MyData" from "MyTable"
				queryReplace = appendRune(queryReplace, r)
				break
			}
			// parameterName will equal a value like "GivenID"
			// from the query string `select "ID" from "MyTable" where "ID" = :GivenID
			parameterName := query[startPos:pos]
			parameters = append(parameters, parameterName)
			switch bindType {
			case bindtype.Question:
				queryReplace = append(queryReplace, '?')
			case bindtype.Named:
				queryReplace = append(queryReplace, ':')
				queryReplace = appendString(queryReplace, parameterName)
			case bindtype.Dollar:
				queryReplace = append(queryReplace, '$')
				queryReplace = appendString(queryReplace, strconv.Itoa(currentParamIndex))
				currentParamIndex++
			case bindtype.At:
				queryReplace = append(queryReplace, '@')
				queryReplace = appendString(queryReplace, strconv.Itoa(currentParamIndex))
				currentParamIndex++
			case bindtype.Unknown:
				if options.sqlXBackwardsCompat {
					queryReplace = append(queryReplace, '?')
				} else {
					return ParseResult{}, errors.New("bind type is not set")
				}
			default:
				return ParseResult{}, errors.New("unhandled bind type: " + bindType.String())
			}
		case '\'', '"':
			startPos := pos
			pos += size

			foundMatch := false
			for pos < len(query) {
				subR, size := utf8.DecodeRuneInString(query[pos:])
				pos += size
				if subR == r {
					foundMatch = true
					break
				}
			}
			if !foundMatch {
				return ParseResult{}, errors.New(`missing matching ` + string(r) + " character between `" + query[startPos:] + "`")
			}
			queryReplace = appendString(queryReplace, query[startPos:pos])
		default:
			pos += size
			queryReplace = appendRune(queryReplace, r)
		}
	}
	var pr ParseResult
	pr.parameters = make([]string, len(parameters))
	for i := range pr.parameters {
		pr.parameters[i] = parameters[i]
	}
	//copy(pr.parameters, parameters)
	pr.query = string(queryReplace)
	return pr, nil
}

func appendRune(slice []byte, run rune) []byte {
	return utf8.AppendRune(slice, run)
}

func appendString(slice []byte, str string) []byte {
	r := slice
	for i := range str {
		r = append(r, str[i])
	}
	return r
}
