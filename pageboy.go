package pageboy

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

type Order string

const (
	ASC  Order = "ASC"
	DESC Order = "DESC"
)

type Comparator string

const (
	GreaterThan Comparator = ">"
	LessThan    Comparator = "<"
)

// CompositeOrder returns order string to specifies the order of the composite key.
//
// Examples:
//
//   CompositeOrder(DESC, "CreatedAt", "ID")
//
func CompositeOrder(order Order, columns ...string) string {
	orders := make([]string, len(columns))
	for i, column := range columns {
		orders[i] = fmt.Sprintf("%s %s", toSnake(column), order)
	}
	return strings.Join(orders, ", ")
}

// CompositeSortScopeFunc returns a function that create a scope for the gorm.
// This scope filters by some position when sorting by composite key.
//
// Examples:
//
//   CompositeSortScopeFunc(GreaterThan, "CreatedAt", "ID")(time, id)
//
func CompositeSortScopeFunc(comparator Comparator, columns ...string) func(values ...interface{}) func(*gorm.DB) *gorm.DB {
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
				switch comparator {
				case LessThan:
					query := fmt.Sprintf("(%s(%s IS NULL OR %s %s ?))", eqQuery, column, column, comparator)
					queries = append(queries, query)
				case GreaterThan:
					query := fmt.Sprintf("(%s%s %s ?)", eqQuery, column, comparator)
					queries = append(queries, query)
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
