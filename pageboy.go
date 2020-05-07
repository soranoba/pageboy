package pageboy

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// CompositeOrder returns order string to specifies the order of the composite key.
//
// Examples:
//
//   CompositeOrder("DESC", "CreatedAt", "ID")
//
func CompositeOrder(sort string, columns ...string) string {
	orders := make([]string, len(columns))
	for i, column := range columns {
		orders[i] = fmt.Sprintf("%s %s", toSnake(column), sort)
	}
	return strings.Join(orders, ", ")
}

// CompositeSortScopeFunc returns a function that create a scope for the gorm.
// This scope filters by some position when sorting by composite key.
//
// Examples:
//
//   CompositeSortScopeFunc(">", "CreatedAt", "ID")(time, id)
//
func CompositeSortScopeFunc(compareStr string, columns ...string) func(values ...interface{}) func(*gorm.DB) *gorm.DB {
	return func(values ...interface{}) func(*gorm.DB) *gorm.DB {
		return func(db *gorm.DB) *gorm.DB {
			queryValues := make([]interface{}, 0)
			queries := make([]string, 0)
			var eqQuery string

			var length = (func() int {
				if len(values) > len(columns) {
					return len(columns)
				}
				return len(values)
			})()

			for i, column := range columns[:length] {
				column = toSnake(column)
				switch compareStr {
				case "<":
					queries = append(queries, fmt.Sprintf("(%s(%s IS NULL OR %s %s ?))", eqQuery, column, column, compareStr))
				case ">":
					queries = append(queries, fmt.Sprintf("(%s%s %s ?)", eqQuery, column, compareStr))
				default:
					panic("Unsupported compareStr")
				}
				queryValues = append(queryValues, values[:i+1]...)
				eqQuery += fmt.Sprintf("%s = ? AND ", column)
			}
			return db.Where(strings.Join(queries, " OR "), queryValues...)
		}
	}
}

func unixToTime(unix float64) *time.Time {
	sec, decimal := math.Modf(unix)
	t := time.Unix(int64(sec), int64(decimal*1e9))
	return &t
}

func toSnake(str string) string {
	runes := []rune(str)
	var p int
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c >= 'A' && c <= 'Z' {
			runes[i] = c - ('A' - 'a')
			if p+1 < i {
				tmp := append([]rune{'_'}, runes[i:]...)
				runes = append(runes[0:i], tmp...)
			}
			p = i
		}
	}
	return string(runes)
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
