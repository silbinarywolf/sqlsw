package bindtype

type Kind int8

// Bindvar types supported by Rebind, BindMap and BindStruct.
const (
	// Unknown means it cannot be determined
	Unknown Kind = 0
	// Question is `$`` parameter type, used by PostgreSQL
	Question Kind = 1
	// Dollar is the `$` parameter type, used by MySQL and SQLite
	Dollar Kind = 2
	// Named is the `:ParamName` parameter type, used by Oracle
	Named Kind = 3
	// At is the `@` parameter type, used by Microsoft SQL Server
	At Kind = 4
)

var bindTypeToName = []string{
	Unknown:  "Unknown",
	Question: "?",
	Dollar:   "$",
	Named:    ":",
	At:       "@",
}

func (kind Kind) String() string {
	return bindTypeToName[kind]
}
