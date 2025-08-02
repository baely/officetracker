package v1

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/report"
)

type Service struct {
	db       database.Databaser
	reporter report.Reporter
	mcp      *mcp.Server
}

func New(db database.Databaser, reporter report.Reporter) *Service {
	s := &Service{
		db:       db,
		reporter: reporter,
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
		Description: "Fetches the users office attendance for the given month. A missing date is functionally equivalent to 'Untracked'",
	}, service.McpGetMonth)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_day",
		Title:       "SetDay",
		Description: "Sets the users office attendance for a given date.",
	}, service.McpSetDay)

	return server
}
