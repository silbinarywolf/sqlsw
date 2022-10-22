package sqlx

// Bindvar types supported by Rebind, BindMap and BindStruct.
const (
	UNKNOWN  = 0
	QUESTION = 1
	DOLLAR   = 2
	NAMED    = 3
	AT       = 4
)
