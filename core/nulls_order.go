package core

// NullsOrder is how to handle null values when sorting.
type NullsOrder int

const (
	// TreatsAsEngineDefault is the default behavior of the engine.
	TreatsAsEngineDefault NullsOrder = iota
	// TreatsAsLowest is that Null values are treated as the lowest.
	TreatsAsLowest
	// TreatsAsHighest is that Null values are treated as the highest.
	TreatsAsHighest
)
