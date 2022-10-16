package sqlparser

import (
	"testing"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
)

const (
	// ignoreCorrectnessInBenchmark can be turned ON to just see how
	// functions perform even if they're broken
	ignoreCorrectnessInBenchmark = false
)

func BenchmarkParse(b *testing.B) {
	i := 1
	testCase := goldenTests[i]
	testCaseNumber := i + 1

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if testCase.Options.BindType == 0 {
			testCase.Options.BindType = bindtype.Question
		}
		r, err := Parse(testCase.Query, testCase.Options)
		if err != nil {
			b.Fatal(err)
		}
		if !ignoreCorrectnessInBenchmark {
			if testCase.ExpectedQuery != r.Query() {
				b.Errorf("test case %d/%d: Query expected to be:\n`%s`\nnot\n`%s`", testCaseNumber, len(goldenTests), testCase.ExpectedQuery, r.Query())
			}
			if len(testCase.ExpectedParameters) != len(r.Parameters()) {
				b.Errorf("test case %d/%d: Parameters expected to be:\n`%+v`\nnot\n`%+v`", testCaseNumber, len(goldenTests), testCase.ExpectedParameters, r.Parameters())
			}
		}
	}
}
