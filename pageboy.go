package pageboy

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type Order string

const (
	ASC  Order = "asc"
	DESC Order = "desc"
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
//   CompositeOrder("CreatedAt", "ID")(DESC, ASC)
//
func CompositeOrder(columns ...string) func(orders ...Order) string {
	return func(orders ...Order) string {
		assert(len(columns) == len(orders), "columns and orders must have the same length")

		parts := make([]string, len(columns))
		for i, column := range columns {
			parts[i] = fmt.Sprintf("%s %s", toSnake(column), strings.ToUpper(string(orders[i])))
		}
		return strings.Join(parts, ", ")
	}
}

func ReverseOrders(orders ...Order) []Order {
	newOrders := make([]Order, len(orders))
	for i := 0; i < len(orders); i++ {
		if orders[i] == ASC {
			newOrders[i] = DESC
		} else {
			newOrders[i] = ASC
		}
	}
	return newOrders
}

// CompositeSortScopeFunc returns a function that create a scope for the gorm.
// This scope filters by some position when sorting by composite key.
//
// Examples:
//
//   CompositeSortScopeFunc("CreatedAt", "ID")(GreaterThan, LessThan)(time, id)
//
func CompositeSortScopeFunc(columns ...string) func(comparators ...Comparator) func(values ...interface{}) func(*gorm.DB) *gorm.DB {
	return func(comparators ...Comparator) func(values ...interface{}) func(*gorm.DB) *gorm.DB {
		assert(len(columns) == len(comparators), "columns and comparators must have the same length")

		return func(values ...interface{}) func(*gorm.DB) *gorm.DB {
			return func(db *gorm.DB) *gorm.DB {
				queryValues := make([]interface{}, 0)
				nonNilValues := make([]interface{}, 0)
				queries := make([]string, 0)
				var eqQuery string

				var length = (func() int {
					if len(values) > len(columns) {
						return len(columns)
					}
					return len(values)
				})()

				comparators = comparators[0:length]
				for i := len(comparators); i < length; i++ {
					if i == 0 {
						comparators = append(comparators, GreaterThan)
					} else {
						comparators = append(comparators, comparators[i-1])
					}
				}

			Loop:
				for i, column := range columns[:length] {
					column = toSnake(column)

					val := reflect.ValueOf(values[i])
					isNil := val.Kind() == reflect.Ptr && val.IsNil()

					switch comparators[i] {
					case LessThan:
						if isNil {
							eqQuery += fmt.Sprintf("%s IS NULL AND ", column)
							continue Loop
						} else {
							query := fmt.Sprintf("(%s(%s IS NULL OR %s %s ?))", eqQuery, column, column, comparators[i])
							queries = append(queries, query)
						}
					case GreaterThan:
						if isNil {
							continue Loop
						} else {
							query := fmt.Sprintf("(%s%s %s ?)", eqQuery, column, comparators[i])
							queries = append(queries, query)
						}
					default:
						panic("Unsupported compareStr")
					}

					eqQuery += fmt.Sprintf("%s = ? AND ", column)
					nonNilValues = append(nonNilValues, values[i])
					queryValues = append(queryValues, nonNilValues...)
				}
				return db.Where("("+strings.Join(queries, " OR ")+")", queryValues...)
			}
		}
	}
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
				i++
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
