package v1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	otctx "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/pkg/model"
)

func (i *Service) McpHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return i.mcp
	}, nil)
}

func (i *Service) McpGetMonth(ctx context.Context, req *mcp.CallToolRequest, in *model.McpGetMonthRequest) (*mcp.CallToolResult, *model.McpGetMonthResponse, error) {
	userID, ok := otctx.MapCtx(ctx).Get(otctx.CtxUserIDKey).(int)
	if !ok {
		return &mcp.CallToolResult{IsError: true}, nil, fmt.Errorf("failed to extract user ID from ctx")
	}

	data, err := i.GetMonth(model.GetMonthRequest{
		Meta: model.GetMonthRequestMeta{
			UserID: userID,
			Year:   in.Year,
			Month:  in.Month,
		},
	})
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, nil, err
	}

	resp := mapGetResp(data)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Officetracker state successfully fetched"},
		},
		IsError: false,
	}, &resp, nil
}

func (i *Service) McpSetDay(ctx context.Context, req *mcp.CallToolRequest, in *model.McpPutDayRequest) (*mcp.CallToolResult, *model.McpPutDayResponse, error) {
	userID, ok := otctx.MapCtx(ctx).Get(otctx.CtxUserIDKey).(int)
	if !ok {
		return &mcp.CallToolResult{IsError: true}, nil, fmt.Errorf("failed to extract user ID from ctx")
	}

	if in == nil {
		return &mcp.CallToolResult{IsError: true}, nil, fmt.Errorf("input is nil")
	}

	putReq, err := mapPutReq(*in)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, nil, err
	}

	putReq.Meta.UserID = userID

	_, err = i.PutDay(putReq)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Officetracker state successfully updated"},
		},
	}, &model.McpPutDayResponse{}, nil
}

func mapGetResp(data model.GetMonthResponse) model.McpGetMonthResponse {
	resp := model.McpGetMonthResponse{
		Dates: []struct {
			Date  int
			State string
		}{},
	}

	for date, state := range data.Data.Days {
		resp.Dates = append(resp.Dates, struct {
			Date  int
			State string
		}{
			Date:  date,
			State: stateToString(state.State),
		})
	}

	return resp
}

func mapPutReq(req model.McpPutDayRequest) (model.PutDayRequest, error) {
	state, err := stateFromString(req.State)
	if err != nil {
		return model.PutDayRequest{}, err
	}

	return model.PutDayRequest{
		Meta: model.PutDayRequestMeta{
			UserID: 0,
			Year:   req.Year,
			Month:  req.Month,
			Day:    req.Date,
		},
		Data: model.DayState{
			State: state,
		},
	}, nil

}

func stateToString(state model.State) string {
	switch state {
	case model.StateUntracked:
		return "Untracked"
	case model.StateWorkFromHome:
		return "WorkFromHome"
	case model.StateWorkFromOffice:
		return "WorkFromOffice"
	case model.StateOther:
		return "Other"
	}
	return "Unknown"
}

func stateFromString(state string) (model.State, error) {
	switch state {
	case "Untracked":
		return model.StateUntracked, nil
	case "WorkFromHome":
		return model.StateWorkFromHome, nil
	case "WorkFromOffice":
		return model.StateWorkFromOffice, nil
	case "Other":
		return model.StateOther, nil
	}
	return 0, fmt.Errorf("Unknown state '%s'. State must be one of 'Untracked', 'WorkFromHome', 'WorkFromOffice' or 'Other'.", state)
}
