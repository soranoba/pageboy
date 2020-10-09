package core

import (
	"fmt"
	"strings"
)

// Order is an ORDER BY clause specified in SQL, and represents the sort order.
type Order string

const (
	// ASC is ascending, with the smallest values first.
	ASC Order = "asc"
	// DESC is descending, with the greatest values first.
	DESC Order = "desc"
)

// OrderClauseBuilder returns a function that create ORDER BY clause to specifies the order of DB records.
func OrderClauseBuilder(columns ...string) func(orders ...string) string {
	return func(orders ...string) string {
		if len(columns) != len(orders) {
			panic("columns and orders must have the same length")
		}

		parts := make([]string, len(columns))
		for i, column := range columns {
			parts[i] = fmt.Sprintf("%s %s", toSnake(column), strings.ToUpper(string(orders[i])))
		}
		return strings.Join(parts, ", ")
	}
}

// ReverseOrders returns a slice of Order converted from ASC to DESC, DESC to ASC, FIRST tO LAST, LAST to FIRST.
func ReverseOrders(orders []string) []string {
	replacer := strings.NewReplacer("ASC", "DESC", "DESC", "ASC", "FIRST", "LAST", "LAST", "FIRST")
	newOrders := make([]string, len(orders))
	for i := 0; i < len(orders); i++ {
		order := strings.ToUpper(orders[i])
		newOrders[i] = replacer.Replace(order)
	}
	return newOrders
}
