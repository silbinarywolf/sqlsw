package sqlxcompat

// Use must be passed into SQLX compatibility functions in sqlsw.
//
// Since this struct is only accessible by packages in this library, it means
// external users cannot use the SQLX compatibility layer.
type Use struct{}
