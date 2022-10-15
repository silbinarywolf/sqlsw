package sqlsw

import (
	"errors"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
)

/* var defaultBinds = map[int][]string{
	DOLLAR:   []string{"postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "ql", "nrpostgres", "cockroach"},
	QUESTION: []string{"mysql", "sqlite3", "nrmysql", "nrsqlite3"},
	NAMED:    []string{"oci8", "ora", "goracle", "godror"},
	AT:       []string{"sqlserver"},
} */

var driverNameToBindType = map[string]bindtype.Kind{
	// Dollar
	"postgres":         bindtype.Dollar,
	"pgx":              bindtype.Dollar,
	"pq-timeouts":      bindtype.Dollar,
	"cloudsqlpostgres": bindtype.Dollar,
	"ql":               bindtype.Dollar,
	"nrpostgres":       bindtype.Dollar,
	"cockroach":        bindtype.Dollar,

	// Question
	"mysql":     bindtype.Question,
	"sqlite3":   bindtype.Question,
	"nrmysql":   bindtype.Question,
	"nrsqlite3": bindtype.Question,

	// Named
	"oci8":    bindtype.Named,
	"ora":     bindtype.Named,
	"goracle": bindtype.Named,
	"godror":  bindtype.Named,

	// At
	"sqlserver": bindtype.At,
}

func RegisterBindType(driverName string, bindType bindtype.Kind) error {
	if _, ok := driverNameToBindType[driverName]; ok {
		return errors.New("cannot bind over existing driver name:" + driverName)
	}
	if bindType == bindtype.Unknown {
		return errors.New(`cannot bind with "Unknown" type`)
	}
	driverNameToBindType[driverName] = bindType
	return nil
}

func getBindTypeFromDriverName(driverName string) (bindtype.Kind, bool) {
	r, ok := driverNameToBindType[driverName]
	return r, ok
}
