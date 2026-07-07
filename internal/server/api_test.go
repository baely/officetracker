package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	context2 "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/pkg/model"
)

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, "bad request", 400)

	res := w.Result()
	if res.StatusCode != 400 {
		t.Errorf("status = %d, want 400", res.StatusCode)
	}
	// The body is a JSON-encoded model.Error carrying the code and message.
	var e model.Error
	if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if e.Code != 400 || e.Message != "bad request" {
		t.Errorf("error body = %+v, want {400, bad request}", e)
	}
	// Note: writeError sets Content-Type application/json, but the following
	// http.Error call resets it to text/plain. Lock in the actual served value.
	if ct := res.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("content-type = %q, want text/plain; charset=utf-8 (http.Error override)", ct)
	}
}

func TestMapResponse(t *testing.T) {
	b, err := mapResponse(model.HealthCheckResponse{Status: "ok"})
	if err != nil {
		t.Fatalf("mapResponse: %v", err)
	}
	if string(b) != `{"status":"ok"}` {
		t.Errorf("mapResponse = %s", b)
	}
}

// getUserID and getAuthMethod read typed values out of the request context.
func TestGetUserIDAndAuthMethod(t *testing.T) {
	val := context2.CtxValue{}
	val.Set(context2.CtxUserIDKey, 8)
	val.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), context2.CtxKey, val))

	if uid, err := getUserID(r); err != nil || uid != 8 {
		t.Errorf("getUserID = (%d, %v), want (8, nil)", uid, err)
	}
	if m, err := getAuthMethod(r); err != nil || m != auth.MethodSSO {
		t.Errorf("getAuthMethod = (%v, %v), want (sso, nil)", m, err)
	}
}

func TestGetUserIDMissing(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil) // no ctx value
	if _, err := getUserID(r); !errors.Is(err, ErrNoUserInCtx) {
		t.Errorf("getUserID error = %v, want ErrNoUserInCtx", err)
	}
}

func TestGetUserIDWrongType(t *testing.T) {
	val := context2.CtxValue{}
	val.Set(context2.CtxUserIDKey, "not-an-int")
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), context2.CtxKey, val))
	if _, err := getUserID(r); !errors.Is(err, ErrNoUserInCtx) {
		t.Errorf("wrong-typed userID should error with ErrNoUserInCtx, got %v", err)
	}
}

// mapRequest populates the user id from context, path params from the chi route,
// and the body from JSON — the full request-decoding pipeline.
func TestMapRequestFullPipeline(t *testing.T) {
	body := `{"data":{"state":2}}`
	r := httptest.NewRequest("PUT", "/state/2024/3/5", strings.NewReader(body))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "3")
	rctx.URLParams.Add("day", "5")

	val := context2.CtxValue{}
	val.Set(context2.CtxUserIDKey, 77)

	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, context2.CtxKey, val)
	r = r.WithContext(ctx)

	req, err := mapRequest[model.PutDayRequest](r)
	if err != nil {
		t.Fatalf("mapRequest: %v", err)
	}
	if req.Meta.UserID != 77 {
		t.Errorf("UserID = %d, want 77 (from context)", req.Meta.UserID)
	}
	if req.Meta.Year != 2024 || req.Meta.Month != 3 || req.Meta.Day != 5 {
		t.Errorf("path params = %+v, want year 2024 month 3 day 5", req.Meta)
	}
	if req.Data.State != model.StateWorkFromOffice {
		t.Errorf("body state = %d, want office", req.Data.State)
	}
}

// When the request has a user_id meta field but no user in context, mapRequest
// fails with an error that unwraps to ErrNoUserInCtx (the 401 signal).
func TestMapRequestMissingUser(t *testing.T) {
	r := httptest.NewRequest("GET", "/state/2024", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	_, err := mapRequest[model.GetYearRequest](r)
	if !errors.Is(err, ErrNoUserInCtx) {
		t.Fatalf("mapRequest error = %v, want to unwrap ErrNoUserInCtx", err)
	}
}

// Query params are decoded via gorilla/schema into schema-tagged fields.
func TestMapRequestQueryParams(t *testing.T) {
	r := httptest.NewRequest("GET", "/report/pdf/2024-attendance?name=Annual", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	val := context2.CtxValue{}
	val.Set(context2.CtxUserIDKey, 1)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, context2.CtxKey, val)
	r = r.WithContext(ctx)

	req, err := mapRequest[model.GetReportRequest](r)
	if err != nil {
		t.Fatalf("mapRequest: %v", err)
	}
	if req.Name != "Annual" {
		t.Errorf("Name = %q, want Annual (from query)", req.Name)
	}
	if req.Meta.Year != 2024 {
		t.Errorf("Year = %d, want 2024", req.Meta.Year)
	}
}
