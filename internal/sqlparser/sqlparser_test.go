package sqlsw

import (
	"testing"
)

func TestParseSelect(t *testing.T) {
	type parseSelectTestCase struct {
		Query    string
		Expected ParseResult
	}
	goldenTests := []parseSelectTestCase{
		/* {
			Query: `select "ID", "Title" from "MyTable" where "ID" = :ID`,
			Expected: ParseResult{
				Query:      `select "ID", "Title" from "MyTable" where "ID" = ?`,
				Parameters: []string{"ID"},
			},
		}, */
		{
			Query: `select "ID", "Title" from "MyTable" where "ID" = :ID and "Title" = 'Hardcoded' and "ID" = :ID`,
			Expected: ParseResult{
				Query:      `select "ID", "Title" from "MyTable" where "ID" = ? and "Title" = 'Hardcoded' and "ID" = ?`,
				Parameters: []string{"ID"},
			},
		},
	}
	for i, testCase := range goldenTests {
		testCaseNumber := i + 1
		r, err := Parse(`select "ID", "Title" from "MyTable" where "ID" = :ID`)
		if err != nil {
			t.Fatal(err)
		}
		if testCase.Expected.Query != r.Query {
			t.Errorf("test case %d/%d: Query expected to be:\n`%s`\nnot\n`%s`", testCaseNumber, len(goldenTests), testCase.Expected.Query, r.Query)
			continue
		}
	}
}
