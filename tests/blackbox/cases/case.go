package cases

import (
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/tests/blackbox/auth"
)

type Endpoint struct {
	Name  string
	Path  string
	Cases []Case
}

type Case struct {
	Name            string
	Method          string
	AuthType        auth.Type
	DataSeed        func(databaser database.Databaser) error
	DataCleanup     func(databaser database.Databaser) error
	RequestPayload  map[string]interface{}
	ResponseStatus  int
	ResponsePayload string
}
