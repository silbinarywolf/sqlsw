package sqlparser

import (
	"testing"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
)

type parseSelectTestCase struct {
	Name                string
	Query               string
	Options             Options
	ExpectedDollarQuery string
	// todo(jae): 2022-10-16
	// test each database driver kind
	// ExpectedQuestionQuery string
	ExpectedParameters []string
}

var goldenTests = []parseSelectTestCase{
	{
		Name:                "Test with one interpolated value, ie. :ID",
		Query:               `select "ID", "Title" from "MyTable" where "ID" = :ID`,
		ExpectedDollarQuery: `select "ID", "Title" from "MyTable" where "ID" = ?`,
		ExpectedParameters:  []string{"ID"},
	},
	{
		Name:                "Test with two of the same interpolated value, ie. two :ID",
		Query:               `select "ID", "Title" from "MyTable" where "ID" = :ID and "Title" = 'Hardcoded' and "ID" = :ID`,
		ExpectedDollarQuery: `select "ID", "Title" from "MyTable" where "ID" = ? and "Title" = 'Hardcoded' and "ID" = ?`,
		ExpectedParameters:  []string{"ID", "ID"},
	},
	{
		Name:                "Test without :",
		Query:               `select "ID", "Title" from "MyTable" where "Title" = 'Hardcoded'`,
		ExpectedDollarQuery: `select "ID", "Title" from "MyTable" where "Title" = 'Hardcoded'`,
		ExpectedParameters:  nil,
	},
	{
		Name:                "Test casting to other type case (PostgreSQL), ie. using :: (double colon)",
		Query:               `select 'SRID=4269;POINT(-123 34)'::geography from "MyTable"`,
		ExpectedDollarQuery: `select 'SRID=4269;POINT(-123 34)'::geography from "MyTable"`,
		ExpectedParameters:  nil,
	},
	{
		Name:                "Test (broken) casting to other type case (PostgreSQL), ie. using :::: (quadruple colon)",
		Query:               `select 'SRID=4269;POINT(-123 34)'::::geography from "MyTable"`,
		ExpectedDollarQuery: `select 'SRID=4269;POINT(-123 34)'::::geography from "MyTable"`,
		ExpectedParameters:  nil,
	},
}

func TestParse(t *testing.T) {
	for i, testCase := range goldenTests {
		if testCase.Options.BindType == 0 {
			testCase.Options.BindType = bindtype.Question
		}
		testCaseNumber := i + 1
		r, err := Parse(testCase.Query, testCase.Options)
		if err != nil {
			t.Fatal(err)
		}
		if testCase.ExpectedDollarQuery != r.Query() {
			t.Errorf("test case %d/%d: Query expected to be:\n`%s`\nnot\n`%s`", testCaseNumber, len(goldenTests), testCase.ExpectedDollarQuery, r.Query())
		}
		if len(testCase.ExpectedParameters) != len(r.Parameters()) {
			t.Errorf("test case %d/%d: Parameters expected to be:\n`%+v`\nnot\n`%+v`", testCaseNumber, len(goldenTests), testCase.ExpectedParameters, r.Parameters())
		}
	}
}
