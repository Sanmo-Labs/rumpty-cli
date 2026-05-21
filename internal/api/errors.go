package api

import "fmt"

type Error struct {
	StatusCode int
	Message    string
	Action     string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("rumpty API error. HTTP %d.", e.StatusCode)
}
