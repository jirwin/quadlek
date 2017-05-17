package tvdb

import (
	"fmt"
)

// JSONError is a generic type for errors
type JSONError struct {
	Message string `json:"error,omitempty"`
}

func (e JSONError) Error() string {
	return fmt.Sprintf("tvdb: %v", e.Message)
}

// Empty checks if an error message is empty
func (e JSONError) Empty() bool {
	if len(e.Message) == 0 {
		return true
	}
	return false
}

// relevantError returns an error or nil
// selects the right error based on the Empty() result
func relevantError(httpError error, jsonError *JSONError) error {
	if httpError != nil {
		return httpError
	}
	if jsonError != nil && !jsonError.Empty() {
		return jsonError
	}
	return nil
}
