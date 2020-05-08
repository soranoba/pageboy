package pageboy

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func TestCompositeOrder(t *testing.T) {
	assertEqual(t, CompositeOrder("ASC", "ID", "CreatedAt"), "id ASC, created_at ASC")
	assertEqual(t, CompositeOrder("DESC", "ID", "CreatedAt"), "id DESC, created_at DESC")
}

func TestUnixToTime(t *testing.T) {
	format := "2006-01-02T15:04:05.999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.250")
	assertNoError(t, err)

	assertEqual(t, *unixToTime(1585706584.25), ti.Local())
}

func TestToSnake(t *testing.T) {
	assertEqual(t, toSnake("CreatedAt"), "created_at")
	assertEqual(t, toSnake("ID"), "id")
	assertEqual(t, toSnake("AbCdEf"), "ab_cd_ef")
}

func openDB() *gorm.DB {
	db, err := gorm.Open(
		"mysql",
		fmt.Sprintf(
			"%s:%s@(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			"pageboy", "pageboy", "127.0.0.1", 3306, "pageboy",
		),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to open a database: %+v", err))
	}
	return db.Debug()
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
