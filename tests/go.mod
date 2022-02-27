module github.com/soranoba/pageboy/tests

go 1.15

require (
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/jackc/pgx/v4 v4.15.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.11 // indirect
	github.com/soranoba/pageboy/v3 v3.0.0
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	gorm.io/driver/mysql v1.3.2
	gorm.io/driver/postgres v1.3.1
	gorm.io/driver/sqlite v1.3.1
	gorm.io/driver/sqlserver v1.3.1
	gorm.io/gorm v1.23.1
)

replace github.com/soranoba/pageboy/v3 => ../
