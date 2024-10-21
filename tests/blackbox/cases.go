package main

import (
	"net/http"

	"github.com/baely/officetracker/tests/blackbox/auth"
)

var cases = []Endpoint{
	{
		Name: "health",
		Path: "/api/v1/health/check",
		Cases: []Case{
			{
				Name:            "success",
				Method:          http.MethodGet,
				AuthType:        auth.NoAuth,
				DataSeed:        nil,
				DataCleanup:     nil,
				RequestPayload:  nil,
				ResponseStatus:  http.StatusOK,
				ResponsePayload: "{\"status\":\"ok\"}",
			},
		},
	},
}
