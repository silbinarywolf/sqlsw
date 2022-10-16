package sqlparser

import (
	"testing"
)

type parseSelectTestCase struct {
	Name               string
	Query              string
	Options            Options
	ExpectedQuery      string
	ExpectedParameters []string
}

var goldenTests = []parseSelectTestCase{
	{
		Name:               "Test with one interpolated value, ie. :ID",
		Query:              `select "ID", "Title" from "MyTable" where "ID" = :ID`,
		ExpectedQuery:      `select "ID", "Title" from "MyTable" where "ID" = ?`,
		ExpectedParameters: []string{"ID"},
	},
	{
		Name:               "Test with two of the same interpolated value, ie. two :ID",
		Query:              `select "ID", "Title" from "MyTable" where "ID" = :ID and "Title" = 'Hardcoded' and "ID" = :ID`,
		ExpectedQuery:      `select "ID", "Title" from "MyTable" where "ID" = ? and "Title" = 'Hardcoded' and "ID" = ?`,
		ExpectedParameters: []string{"ID", "ID"},
	},
	{
		Name:               "Test without :",
		Query:              `select "ID", "Title" from "MyTable" where "Title" = 'Hardcoded'`,
		ExpectedQuery:      `select "ID", "Title" from "MyTable" where "Title" = 'Hardcoded'`,
		ExpectedParameters: nil,
	},
	{
		Name:               "Test casting to other type case (PostgreSQL), ie. using :: (double colon)",
		Query:              `select 'SRID=4269;POINT(-123 34)'::geography from "MyTable"`,
		ExpectedQuery:      `select 'SRID=4269;POINT(-123 34)'::geography from "MyTable"`,
		ExpectedParameters: nil,
	},
	{
		Name:               "Test (broken) casting to other type case (PostgreSQL), ie. using :::: (quadruple colon)",
		Query:              `select 'SRID=4269;POINT(-123 34)'::::geography from "MyTable"`,
		ExpectedQuery:      `select 'SRID=4269;POINT(-123 34)'::::geography from "MyTable"`,
		ExpectedParameters: nil,
	},
}

func TestParseSelect(t *testing.T) {
	for i, testCase := range goldenTests {
		testCaseNumber := i + 1
		r, err := Parse(testCase.Query, testCase.Options)
		if err != nil {
			t.Fatal(err)
		}
		if testCase.ExpectedQuery != r.Query() {
			t.Errorf("test case %d/%d: Query expected to be:\n`%s`\nnot\n`%s`", testCaseNumber, len(goldenTests), testCase.ExpectedQuery, r.Query())
		}
		if len(testCase.ExpectedParameters) != len(r.Parameters()) {
			t.Errorf("test case %d/%d: Parameters expected to be:\n`%+v`\nnot\n`%+v`", testCaseNumber, len(goldenTests), testCase.ExpectedParameters, r.Parameters())
		}
	}
}
