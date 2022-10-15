package sqlsw

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

type Parameter struct {
	Name string
}

type ParseResult struct {
	Query      string
	Parameters []string

	parametersUnderlying [8]string
}

// Parse will
func Parse(query string) (ParseResult, error) {
	var (
		pr         ParseResult
		stackBytes [128]byte
	)

	// Setup query replacement buffer
	var queryReplace []byte
	if len(query) > len(stackBytes) {
		queryReplace = make([]byte, 0, len(query))
	} else {
		queryReplace = stackBytes[:0]
	}
	parameters := pr.parametersUnderlying[:0]
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
