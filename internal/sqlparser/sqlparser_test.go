package sqlparser

import (
	"testing"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
)

type parseTestCase struct {
	Name               string
	Query              string
	Options            Options
	ExpectedQuery      string
	ExpectedParameters []string
}

var goldenTests = []parseTestCase{
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
	{
		Name:               "Test : within string (\")",
		Query:              `select ":ID" from "MyTable"`,
		ExpectedQuery:      `select ":ID" from "MyTable"`,
		ExpectedParameters: nil,
	},
	{
		Name:               "Test : within string (')",
		Query:              `select ':ID' from "MyTable"`,
		ExpectedQuery:      `select ':ID' from "MyTable"`,
		ExpectedParameters: nil,
	},
	{
		Name:               "Test : within string (`)",
		Query:              "select `:ID` from my_table",
		ExpectedQuery:      "select `:ID` from my_table",
		ExpectedParameters: nil,
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
		if expected := testCase.ExpectedQuery; expected != r.Query() {
			t.Errorf("test case %d/%d: Query expected to be:\n`%s`\nnot\n`%s`", testCaseNumber, len(goldenTests), expected, r.Query())
		}
		if len(testCase.ExpectedParameters) != len(r.Parameters()) {
			t.Errorf("test case %d/%d: Parameters expected to be:\n`%+v`\nnot\n`%+v`", testCaseNumber, len(goldenTests), testCase.ExpectedParameters, r.Parameters())
		}
	}
}

func TestParseBindType(t *testing.T) {
	type parseBindTypeTestCase struct {
		Name                  string
		Query                 string
		Options               Options
		ExpectedQuestionQuery string
		// todo(jae): 2022-10-16
		// test each database driver kind
		// ExpectedDollarQuery string
		ExpectedParameters []string
	}
	var goldenTests = []parseBindTypeTestCase{
		{
			Name:                  "Test with one interpolated value, ie. :ID",
			Query:                 `select "ID", "Title" from "MyTable" where "ID" = :ID`,
			ExpectedQuestionQuery: `select "ID", "Title" from "MyTable" where "ID" = ?`,
			ExpectedParameters:    []string{"ID"},
		},
	}
	for i, testCase := range goldenTests {
		if testCase.Options.BindType == 0 {
			testCase.Options.BindType = bindtype.Question
		}
		testCaseNumber := i + 1
		r, err := Parse(testCase.Query, testCase.Options)
		if err != nil {
			t.Fatal(err)
		}
		if expected := testCase.ExpectedQuestionQuery; expected != r.Query() {
			t.Errorf("test case %d/%d: Query expected to be:\n`%s`\nnot\n`%s`", testCaseNumber, len(goldenTests), expected, r.Query())
		}
		if len(testCase.ExpectedParameters) != len(r.Parameters()) {
			t.Errorf("test case %d/%d: Parameters expected to be:\n`%+v`\nnot\n`%+v`", testCaseNumber, len(goldenTests), testCase.ExpectedParameters, r.Parameters())
		}
	}
}
