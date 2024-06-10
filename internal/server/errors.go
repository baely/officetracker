package server

import "fmt"

const (
	internalErrorMsg = "Internal server error"
)

var (
	ErrNoUserInCtx = fmt.Errorf("user ID not in context")
)
