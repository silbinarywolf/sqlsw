package sqlx

import (
	"database/sql"
	"fmt"
	"testing"
)

type Test struct {
	t *testing.T
}

func (t Test) Error(err error, msg ...interface{}) {
	t.t.Helper()
	if err != nil {
		if len(msg) == 0 {
			t.t.Error(err)
		} else {
			t.t.Error(msg...)
		}
	}
}

func (t Test) Errorf(err error, format string, args ...interface{}) {
	t.t.Helper()
	if err != nil {
		t.t.Errorf(format, args...)
	}
}
