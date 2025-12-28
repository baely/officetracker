package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otctx "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestStateToString(t *testing.T) {
	tests := []struct {
		name  string
		state model.State
		want  string
	}{
		{
			name:  "untracked",
			state: model.StateUntracked,
			want:  "Untracked",
		},
		{
			name:  "work from home",
			state: model.StateWorkFromHome,
			want:  "WorkFromHome",
		},
		{
			name:  "work from office",
			state: model.StateWorkFromOffice,
			want:  "WorkFromOffice",
		},
		{
			name:  "other",
			state: model.StateOther,
			want:  "Other",
		},
		{
			name:  "unknown state",
			state: model.State(99),
			want:  "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stateToString(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStateFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    model.State
		wantErr bool
	}{
		{
			name:  "untracked",
			input: "Untracked",
			want:  model.StateUntracked,
		},
		{
			name:  "work from home",
			input: "WorkFromHome",
			want:  model.StateWorkFromHome,
		},
		{
			name:  "work from office",
			input: "WorkFromOffice",
			want:  model.StateWorkFromOffice,
		},
		{
			name:  "other",
			input: "Other",
			want:  model.StateOther,
		},
		{
			name:    "unknown string",
			input:   "InvalidState",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stateFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMapGetResp(t *testing.T) {
	input := model.GetMonthResponse{
		Data: model.MonthState{
			Days: map[int]model.DayState{
				1: {State: model.StateWorkFromHome},
				15: {State: model.StateWorkFromOffice},
				30: {State: model.StateOther},
			},
		},
	}

	resp := mapGetResp(input)

	// Verify all dates are included
	assert.Len(t, resp.Dates, 3)

	// Create a map for easier verification
	dateMap := make(map[int]string)
	for _, d := range resp.Dates {
		dateMap[d.Date] = d.State
	}

	assert.Equal(t, "WorkFromHome", dateMap[1])
	assert.Equal(t, "WorkFromOffice", dateMap[15])
	assert.Equal(t, "Other", dateMap[30])
}

func TestMapGetResp_EmptyDays(t *testing.T) {
	input := model.GetMonthResponse{
		Data: model.MonthState{
			Days: map[int]model.DayState{},
		},
	}

	resp := mapGetResp(input)

	assert.Empty(t, resp.Dates)
}

func TestMapPutReq(t *testing.T) {
	input := model.McpPutDayRequest{
		Year:  2024,
		Month: 10,
		Date:  15,
		State: "WorkFromOffice",
	}

	result, err := mapPutReq(input)
	require.NoError(t, err)

	assert.Equal(t, 2024, result.Meta.Year)
	assert.Equal(t, 10, result.Meta.Month)
	assert.Equal(t, 15, result.Meta.Day)
	assert.Equal(t, 0, result.Meta.UserID) // UserID is always 0 for MCP
	assert.Equal(t, model.StateWorkFromOffice, result.Data.State)
}

func TestMapPutReq_InvalidState(t *testing.T) {
	input := model.McpPutDayRequest{
		Year:  2024,
		Month: 10,
		Date:  15,
		State: "InvalidState",
	}

	_, err := mapPutReq(input)
	assert.Error(t, err)
}


func TestMcpHandler(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	handler := service.McpHandler()
	assert.NotNil(t, handler)
}

func TestMcpGetMonth(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.SetDay(1, 15, 12, 2024, model.DayState{State: model.StateWorkFromHome})
	
	service := New(mockDB, nil)

	ctx := context.Background()
	ctxVal := otctx.CtxValue{}
	ctxVal.Set(otctx.CtxUserIDKey, 1)
	ctx = context.WithValue(ctx, otctx.CtxKey, ctxVal)

	result, resp, err := service.McpGetMonth(ctx, nil, &model.McpGetMonthRequest{
		Year:  2024,
		Month: 12,
	})

	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Dates)
}

func TestMcpGetMonth_NoUserID(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	ctx := context.Background()

	result, resp, err := service.McpGetMonth(ctx, nil, &model.McpGetMonthRequest{
		Year:  2024,
		Month: 12,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, result.IsError)
}

func TestMcpSetDay(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	ctx := context.Background()
	ctxVal := otctx.CtxValue{}
	ctxVal.Set(otctx.CtxUserIDKey, 1)
	ctx = context.WithValue(ctx, otctx.CtxKey, ctxVal)

	result, resp, err := service.McpSetDay(ctx, nil, &model.McpPutDayRequest{
		Year:  2024,
		Month: 12,
		Date:  15,
		State: "WorkFromHome",
	})

	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotNil(t, resp)
}

func TestMcpSetDay_NoUserID(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	ctx := context.Background()

	result, resp, err := service.McpSetDay(ctx, nil, &model.McpPutDayRequest{
		Year:  2024,
		Month: 12,
		Date:  15,
		State: "WorkFromHome",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, result.IsError)
}
