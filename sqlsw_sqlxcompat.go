package sqlsw

import (
	"database/sql"

	"github.com/silbinarywolf/sqlsw/internal/bindtype"
)

// --------
// WARNING:
// --------
// This file contains any SQLX compatbility layer functions
// None of this are guaranteed to work in the future so do not use this.

// SQLX_CompatNewDB exists to support NewDb in the sqlx backwards compatibility driver.
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_CompatNewDB(db *sql.DB, driverName string) (*DB, error) {
	dbWrapper, err := newDB(db, driverName)
	return dbWrapper, err
}

// SQLX_GetBindType exists to support NewDb in the sqlx backwards compatibility driver.
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_GetBindType(db *DB) bindtype.Kind {
	return db.bindType
}

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

// SQLX_Rows_From_Row returns the underlying "database/sql" handle
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_Rows_From_Row(row *Row) *sql.Rows { return row.rows.rows }

// SQLX_NamedStmt returns the underlying "database/sql" handle
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_NamedStmt(namedStmt *NamedStmt) *sql.Stmt { return namedStmt.underlying }

// SQLX_NewRows creates a Rows struct
//
// Deprecated: This may be changed or removed in the future. Do not use.
func SQLX_NewRows(rows *sql.Rows, cachingData cachingObject) *Rows {
	return newRows(rows, cachingData.getCachingData())
}
