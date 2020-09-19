package core

import (
	"fmt"
)

func ExampleOrderClauseBuilder() {
	fmt.Printf("%s\n", OrderClauseBuilder("ID", "CreatedAt")(ASC, ASC))
	fmt.Printf("%s\n", OrderClauseBuilder("ID", "CreatedAt")(DESC, DESC))
	fmt.Printf("%s\n", OrderClauseBuilder("ID", "CreatedAt")(ASC, DESC))

	// Output:
	// `id` ASC, `created_at` ASC
	// `id` DESC, `created_at` DESC
	// `id` ASC, `created_at` DESC
}

func ExampleReverseOrders() {
	fmt.Printf("%v\n", ReverseOrders([]Order{ASC, DESC, DESC}))
	fmt.Printf("%v\n", ReverseOrders([]Order{DESC, ASC, ASC}))

	// Output:
	// [desc asc asc]
	// [asc desc desc]
}
