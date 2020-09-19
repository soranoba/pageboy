package pageboy

import "fmt"

// ValidationError is a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("%s %s", err.Field, err.Message)
}
