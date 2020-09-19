package core

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// Comparison is a comparison operator used by where clause in SQL.
type Comparison string

const (
	// GreaterThan means that the column value in DB record is greater than the specified value.
	GreaterThan Comparison = ">"
	// LessThan means that the column value in DB record is less than the specified value.
	LessThan Comparison = "<"
)

// MakeComparisonScopeBuildFunc returns a GORM scope builder.
// This scope add a where clauses filtered by comparisons ranges.
func MakeComparisonScopeBuildFunc(columns ...string) func(comparisions ...Comparison) func(values ...interface{}) func(*gorm.DB) *gorm.DB {
	return func(comparisions ...Comparison) func(values ...interface{}) func(*gorm.DB) *gorm.DB {
		if len(columns) != len(comparisions) {
			panic("columns and comparisions must have the same length")
		}

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

				comparisions = comparisions[0:length]
				for i := len(comparisions); i < length; i++ {
					if i == 0 {
						comparisions = append(comparisions, GreaterThan)
					} else {
						comparisions = append(comparisions, comparisions[i-1])
					}
				}

			Loop:
				for i, column := range columns[:length] {
					column = toSnake(column)

					val := reflect.ValueOf(values[i])
					isNil := val.Kind() == reflect.Ptr && val.IsNil()

					switch comparisions[i] {
					case LessThan:
						if isNil {
							eqQuery += fmt.Sprintf("`%s` IS NULL AND ", column)
							continue Loop
						} else {
							query := fmt.Sprintf("(%s(`%s` IS NULL OR `%s` %s ?))", eqQuery, column, column, comparisions[i])
							queries = append(queries, query)
						}
					case GreaterThan:
						if isNil {
							eqQuery += fmt.Sprintf("`%s` IS NOT NULL OR ", column)
							continue Loop
						} else {
							query := fmt.Sprintf("(%s`%s` %s ?)", eqQuery, column, comparisions[i])
							queries = append(queries, query)
						}
					default:
						panic("Unsupported compareStr")
					}

					eqQuery += fmt.Sprintf("`%s` = ? AND ", column)
					nonNilValues = append(nonNilValues, values[i])
					queryValues = append(queryValues, nonNilValues...)
				}
				return db.Where("("+strings.Join(queries, " OR ")+")", queryValues...)
			}
		}
	}
}