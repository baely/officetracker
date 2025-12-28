package v1

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/internal/util"
)

type Service struct {
	db       database.Databaser
	reporter report.Reporter
	mcp      *mcp.Server
	clock    util.Clock
}

func New(db database.Databaser, reporter report.Reporter) *Service {
	return NewWithClock(db, reporter, &util.RealClock{})
}

// NewWithClock creates a Service with a custom clock (useful for testing)
func NewWithClock(db database.Databaser, reporter report.Reporter, clock util.Clock) *Service {
	s := &Service{
		db:       db,
		reporter: reporter,
		clock:    clock,
	}

	s.mcp = createMcpServer(s)

	return s
}

func createMcpServer(service *Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "officetracker",
		Title:   "Officetracker",
		Version: "0.0.1",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_month",
		Title:       "GetMonth",
		Description: "Fetches the users office attendance for the given month. A missing date is functionally equivalent to 'Untracked' which is to say the user didn't state their office attendance.",
	}, service.McpGetMonth)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_day",
		Title:       "SetDay",
		Description: "Sets the users office attendance for a given date. Valid states are 'Untracked', 'WorkFromHome', 'WorkFromOffice' or 'Other'.",
	}, service.McpSetDay)

	return server
}
