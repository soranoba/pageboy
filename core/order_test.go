package core

import (
	"fmt"
)

func ExampleOrderClauseBuilder() {
	fmt.Printf("%s\n", OrderClauseBuilder("ID", "CreatedAt")("asc", "ASC"))
	fmt.Printf("%s\n", OrderClauseBuilder("ID", "CreatedAt")("desc", "DESC"))
	fmt.Printf("%s\n", OrderClauseBuilder("ID", "CreatedAt")("ASC", "desc"))

	// Output:
	// id ASC, created_at ASC
	// id DESC, created_at DESC
	// id ASC, created_at DESC
}

func ExampleReverseOrders() {
	fmt.Printf("%#v\n", ReverseOrders([]string{"asc", "DESC", "desc nulls first"}))
	fmt.Printf("%#v\n", ReverseOrders([]string{"DESC", "asc", "ASC NULLS LAST"}))

	// Output:
	// []string{"DESC", "ASC", "ASC NULLS LAST"}
	// []string{"ASC", "DESC", "DESC NULLS FIRST"}
}
