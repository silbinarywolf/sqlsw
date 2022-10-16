package sqlsw

import (
	"github.com/silbinarywolf/sqlsw/internal/bindtype"
	"github.com/silbinarywolf/sqlsw/internal/dbreflect"
)

func newDB() *DB {
	db := &DB{}
	db.bindType = bindtype.Question
	db.reflector = &dbreflect.ReflectModule{}
	return db
}

// todo(jae): 2022-10-16
// test transform to list of parameters
/* func TestAFjak(t *testing.T) {
	db := newDB()
	type structInfo struct {
		ID int64 `db:"ID"`
	}
	q := `select "ID" from "MyTable" where "ID" = :ID`
	query, argList, err := transformNamedQueryAndParams(db.reflector, db.bindType, q, structInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if len(argList) == 0 {
		t.Fatal("arg list should be more")
	}
	panic(query)
}
*/
