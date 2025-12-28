package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapCtx(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want CtxValue
	}{
		{
			name: "context with CtxValue",
			ctx: context.WithValue(context.Background(), CtxKey, CtxValue{
				"userID": 123,
				"auth":   "sso",
			}),
			want: CtxValue{
				"userID": 123,
				"auth":   "sso",
			},
		},
		{
			name: "context without CtxValue",
			ctx:  context.Background(),
			want: CtxValue{},
		},
		{
			name: "context with wrong type",
			ctx:  context.WithValue(context.Background(), CtxKey, "wrong type"),
			want: CtxValue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapCtx(tt.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetCtxValue(t *testing.T) {
	tests := []struct {
		name string
		req  *http.Request
		want CtxValue
	}{
		{
			name: "request with context value",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				ctx := context.WithValue(req.Context(), CtxKey, CtxValue{
					"userID": 456,
					"auth":   "secret",
				})
				return req.WithContext(ctx)
			}(),
			want: CtxValue{
				"userID": 456,
				"auth":   "secret",
			},
		},
		{
			name: "request without context value",
			req:  httptest.NewRequest("GET", "/", nil),
			want: CtxValue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCtxValue(tt.req)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCtxValue_Set(t *testing.T) {
	tests := []struct {
		name     string
		initial  CtxValue
		key      string
		val      interface{}
		expected CtxValue
	}{
		{
			name:    "set value in empty context",
			initial: CtxValue{},
			key:     "userID",
			val:     789,
			expected: CtxValue{
				"userID": 789,
			},
		},
		{
			name: "set value in existing context",
			initial: CtxValue{
				"existing": "value",
			},
			key: "userID",
			val: 999,
			expected: CtxValue{
				"existing": "value",
				"userID":   999,
			},
		},
		{
			name: "overwrite existing value",
			initial: CtxValue{
				"userID": 111,
			},
			key: "userID",
			val: 222,
			expected: CtxValue{
				"userID": 222,
			},
		},
		{
			name:    "set string value",
			initial: CtxValue{},
			key:     "auth",
			val:     "sso",
			expected: CtxValue{
				"auth": "sso",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.initial.Set(tt.key, tt.val)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCtxValue_Get(t *testing.T) {
	tests := []struct {
		name     string
		ctxValue CtxValue
		key      string
		want     interface{}
	}{
		{
			name: "get existing int value",
			ctxValue: CtxValue{
				"userID": 123,
			},
			key:  "userID",
			want: 123,
		},
		{
			name: "get existing string value",
			ctxValue: CtxValue{
				"auth": "sso",
			},
			key:  "auth",
			want: "sso",
		},
		{
			name: "get non-existing value",
			ctxValue: CtxValue{
				"userID": 123,
			},
			key:  "nonexistent",
			want: nil,
		},
		{
			name:     "get from empty context",
			ctxValue: CtxValue{},
			key:      "anykey",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ctxValue.Get(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCtxValue_SetAndGet(t *testing.T) {
	// Test chaining Set and Get
	ctx := CtxValue{}

	// Set userID
	ctx = ctx.Set(CtxUserIDKey, 123)
	assert.Equal(t, 123, ctx.Get(CtxUserIDKey))

	// Set auth method
	ctx = ctx.Set(CtxAuthMethodKey, "sso")
	assert.Equal(t, "sso", ctx.Get(CtxAuthMethodKey))

	// Verify both values exist
	assert.Equal(t, 123, ctx.Get(CtxUserIDKey))
	assert.Equal(t, "sso", ctx.Get(CtxAuthMethodKey))
}

func TestContextConstants(t *testing.T) {
	// Verify constants have expected values
	assert.Equal(t, "ctx", CtxKey)
	assert.Equal(t, "userID", CtxUserIDKey)
	assert.Equal(t, "auth", CtxAuthMethodKey)
}
