module github.com/silbinarywolf/sqlsw/sqlx

go 1.16

replace github.com/silbinarywolf/sqlsw => ../

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/silbinarywolf/sqlsw v0.0.0-00010101000000-000000000000
)
