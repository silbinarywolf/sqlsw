package sqlsw

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

type Options struct {
}

type ParseResult struct {
	Query      string
	Parameters []string

	// parametersUnderlyingData will reduce allocs for small interpolation
	// cases
	parametersUnderlyingData [4]string
}

// Parse will
func Parse(query string, options Options) (ParseResult, error) {
	var (
		pr         ParseResult
		stackBytes [256]byte
	)
	// Setup query replacement buffer
	var queryReplace []byte
	if len(query) > len(stackBytes) {
		queryReplace = make([]byte, 0, len(query))
	} else {
		// Use bytes on the stack while building the new query string
		queryReplace = stackBytes[:0]
	}
	parameters := pr.parametersUnderlyingData[:0]
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
			name := query[startPos:pos]
			parameters = append(parameters, name)
			queryReplace = append(queryReplace, '?')
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
				return pr, errors.New(`missing matching ` + string(r) + " character between `" + query[startPos:] + "`")
			}
			queryReplace = appendString(queryReplace, query[startPos:pos])
		default:
			pos += size
			queryReplace = appendRune(queryReplace, r)
		}
	}
	pr.Query = string(queryReplace)
	pr.Parameters = parameters
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
