package v1

import (
	"context"
	"testing"

	otctx "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/pkg/model"
)

func TestStateStringRoundTrip(t *testing.T) {
	cases := []struct {
		state model.State
		str   string
	}{
		{model.StateUntracked, "Untracked"},
		{model.StateWorkFromHome, "WorkFromHome"},
		{model.StateWorkFromOffice, "WorkFromOffice"},
		{model.StateOther, "Other"},
	}
	for _, c := range cases {
		if got := stateToString(c.state); got != c.str {
			t.Errorf("stateToString(%d) = %q, want %q", c.state, got, c.str)
		}
		back, err := stateFromString(c.str)
		if err != nil || back != c.state {
			t.Errorf("stateFromString(%q) = (%d, %v), want (%d, nil)", c.str, back, err, c.state)
		}
	}

	// Scheduled states have no string form.
	if got := stateToString(model.StateScheduledWorkFromOffice); got != "Unknown" {
		t.Errorf("stateToString(scheduled) = %q, want Unknown", got)
	}
	// An unrecognised string is rejected.
	if _, err := stateFromString("Teleporting"); err == nil {
		t.Error("stateFromString should reject unknown states")
	}
}

func TestMapPutReq(t *testing.T) {
	got, err := mapPutReq(model.McpPutDayRequest{Year: 2024, Month: 3, Date: 5, State: "WorkFromOffice"})
	if err != nil {
		t.Fatalf("mapPutReq: %v", err)
	}
	if got.Meta.Year != 2024 || got.Meta.Month != 3 || got.Meta.Day != 5 {
		t.Errorf("mapPutReq meta = %+v", got.Meta)
	}
	if got.Data.State != model.StateWorkFromOffice {
		t.Errorf("mapPutReq state = %d, want office", got.Data.State)
	}

	if _, err := mapPutReq(model.McpPutDayRequest{State: "bogus"}); err == nil {
		t.Error("mapPutReq should reject an invalid state string")
	}
}

func TestMapGetResp(t *testing.T) {
	resp := mapGetResp(model.GetMonthResponse{Data: model.MonthState{Days: map[int]model.DayState{
		1: {State: model.StateWorkFromOffice},
		2: {State: model.StateWorkFromHome},
	}}})
	if len(resp.Dates) != 2 {
		t.Fatalf("got %d dates, want 2", len(resp.Dates))
	}
	byDate := map[int]string{}
	for _, d := range resp.Dates {
		byDate[d.Date] = d.State
	}
	if byDate[1] != "WorkFromOffice" || byDate[2] != "WorkFromHome" {
		t.Errorf("mapGetResp dates = %v", byDate)
	}
}

func ctxWithUser(userID int) context.Context {
	val := otctx.CtxValue{}
	val.Set(otctx.CtxUserIDKey, userID)
	return context.WithValue(context.Background(), otctx.CtxKey, val)
}

// McpGetMonth resolves the user from context and returns their month's states.
func TestMcpGetMonth(t *testing.T) {
	db := dbtest.New()
	db.SaveDay(1, 10, 3, 2024, model.DayState{State: model.StateWorkFromOffice})
	svc := &Service{db: db}

	_, out, err := svc.McpGetMonth(ctxWithUser(1), nil, &model.McpGetMonthRequest{Year: 2024, Month: 3})
	if err != nil {
		t.Fatalf("McpGetMonth: %v", err)
	}
	if len(out.Dates) != 1 || out.Dates[0].Date != 10 || out.Dates[0].State != "WorkFromOffice" {
		t.Errorf("McpGetMonth result = %+v", out.Dates)
	}
}

func TestMcpGetMonthNoUser(t *testing.T) {
	svc := &Service{db: dbtest.New()}
	res, _, err := svc.McpGetMonth(context.Background(), nil, &model.McpGetMonthRequest{})
	if err == nil {
		t.Fatal("expected error when user id absent from context")
	}
	if res == nil || !res.IsError {
		t.Error("expected an error CallToolResult")
	}
}

// McpSetDay resolves the user from context and writes the day.
func TestMcpSetDay(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	_, _, err := svc.McpSetDay(ctxWithUser(1), nil, &model.McpPutDayRequest{
		Year: 2024, Month: 3, Date: 12, State: "WorkFromHome",
	})
	if err != nil {
		t.Fatalf("McpSetDay: %v", err)
	}
	got, _ := db.GetDay(1, 12, 3, 2024)
	if got.State != model.StateWorkFromHome {
		t.Errorf("day not written: %d", got.State)
	}
}

func TestMcpSetDayInvalidState(t *testing.T) {
	svc := &Service{db: dbtest.New()}
	res, _, err := svc.McpSetDay(ctxWithUser(1), nil, &model.McpPutDayRequest{State: "nope"})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if res == nil || !res.IsError {
		t.Error("expected an error CallToolResult")
	}
}

func TestMcpSetDayNoUser(t *testing.T) {
	svc := &Service{db: dbtest.New()}
	if _, _, err := svc.McpSetDay(context.Background(), nil, &model.McpPutDayRequest{State: "Other"}); err == nil {
		t.Fatal("expected error when user id absent from context")
	}
}
