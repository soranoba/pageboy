package pageboy

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openDB() *gorm.DB {
	var (
		db  *gorm.DB
		err error
	)

	dbType, _ := os.LookupEnv("db")
	switch dbType {
	case "mysql":
		db, err = gorm.Open(
			mysql.Open(
				fmt.Sprintf(
					"%s:%s@(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
					"pageboy", "pageboy", "127.0.0.1", 3306, "pageboy",
				),
			),
			&gorm.Config{},
		)
	default:
		dir, err := ioutil.TempDir("", "pageboy_*")
		if err != nil {
			panic(fmt.Sprintf("failed to create tmp dir: %+v", err))
		}
		db, err = gorm.Open(
			sqlite.Open(path.Join(dir, "test.db")),
			&gorm.Config{},
		)
	}

	if err != nil {
		panic(fmt.Sprintf("failed to open a database: %+v", err))
	}
	return db
}

func assertEqual(t *testing.T, got, expected interface{}) bool {
	if !reflect.DeepEqual(got, expected) {
		_, file, line, _ := runtime.Caller(1)
		t.Errorf("Not equals:\n  file    : %s:%d\n  got     : %#v\n  expected: %#v\n", file, line, got, expected)
		return false
	}
	return true
}

func assertNotEqual(t *testing.T, got, expected interface{}) bool {
	if reflect.DeepEqual(got, expected) {
		_, file, line, _ := runtime.Caller(1)
		t.Errorf("Equals:\n  file    : %s:%d\n  got     : %#v\n", file, line, got)
		return false
	}
	return true
}

func assertError(t *testing.T, err error) bool {
	if err == nil {
		_, file, line, _ := runtime.Caller(1)
		t.Errorf("NoError:\n  file    : %s:%d\n  ", file, line)
		return false
	}
	return true
}

func assertNoError(t *testing.T, err error) bool {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		t.Errorf("Error:\n  file    : %s:%d\n  error   : %#vn", file, line, err)
		return false
	}
	return true
}
