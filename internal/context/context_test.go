package context

import (
	"context"
	"net/http/httptest"
	"testing"
)

// A CtxValue stored under CtxKey round-trips through MapCtx and Get.
func TestMapCtxAndGet(t *testing.T) {
	val := CtxValue{}
	val.Set(CtxUserIDKey, 42)
	val.Set(CtxAuthMethodKey, "sso")

	ctx := context.WithValue(context.Background(), CtxKey, val)
	got := MapCtx(ctx)

	if got.Get(CtxUserIDKey) != 42 {
		t.Errorf("userID = %v, want 42", got.Get(CtxUserIDKey))
	}
	if got.Get(CtxAuthMethodKey) != "sso" {
		t.Errorf("auth = %v, want sso", got.Get(CtxAuthMethodKey))
	}
}

// MapCtx returns an empty (non-nil) CtxValue when nothing is stored, so callers
// can Get without a nil-map panic.
func TestMapCtxMissing(t *testing.T) {
	got := MapCtx(context.Background())
	if got == nil {
		t.Fatal("MapCtx returned nil map")
	}
	if v := got.Get("anything"); v != nil {
		t.Errorf("Get on empty ctx = %v, want nil", v)
	}
}

// MapCtx ignores a value stored under CtxKey with the wrong type.
func TestMapCtxWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKey, "not a CtxValue")
	got := MapCtx(ctx)
	if len(got) != 0 {
		t.Errorf("expected empty CtxValue for wrong-typed value, got %v", got)
	}
}

func TestGetCtxValueFromRequest(t *testing.T) {
	val := CtxValue{}
	val.Set(CtxUserIDKey, 7)
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), CtxKey, val))

	if GetCtxValue(r).Get(CtxUserIDKey) != 7 {
		t.Errorf("GetCtxValue userID = %v, want 7", GetCtxValue(r).Get(CtxUserIDKey))
	}
}

// Set mutates and returns the same underlying map (chaining contract).
func TestSetReturnsReceiver(t *testing.T) {
	val := CtxValue{}
	returned := val.Set("k", "v")
	returned.Set("k2", "v2")
	if val.Get("k") != "v" || val.Get("k2") != "v2" {
		t.Errorf("Set did not mutate the shared map: %v", val)
	}
}
