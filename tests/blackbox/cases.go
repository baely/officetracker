package main

import (
	cases2 "github.com/baely/officetracker/tests/blackbox/cases"
	"github.com/baely/officetracker/tests/blackbox/cases/health"
)

var cases = []cases2.Endpoint{
	health.Check,
}
