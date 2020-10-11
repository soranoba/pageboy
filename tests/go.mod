module github.com/soranoba/pageboy/tests

go 1.15

require (
	github.com/soranoba/pageboy v1.1.0
	gorm.io/driver/mysql v1.0.2
	gorm.io/driver/postgres v1.0.2
	gorm.io/driver/sqlite v1.1.3
	gorm.io/driver/sqlserver v1.0.4
	gorm.io/gorm v1.20.2
)

replace github.com/soranoba/pageboy => ../
