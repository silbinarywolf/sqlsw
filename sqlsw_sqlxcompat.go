package sqlsw

import (
	"database/sql"
)

// --------
// WARNING:
// --------
// This file contains any SQLX compatbility layer functions
// None of this are guaranteed to work in the future so do not use this.

// SQLX_DB returns the underlying "database/sql" handle
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_DB(db *DB) *sql.DB { return db.db }

// SQLX_Tx returns the underlying "database/sql" handle
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_Tx(tx *Tx) *sql.Tx { return tx.underlying }

// SQLX_Rows returns the underlying "database/sql" handle
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_Rows(rows *Rows) *sql.Rows { return rows.rows }

// SQLX_NewRows creates a Rows struct
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_NewRows(rows *sql.Rows, cachingData cachingObject) *Rows {
	return newRows(rows, cachingData.getCachingData())
}
