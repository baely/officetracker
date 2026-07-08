package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database/dbtest"
)

// When no native client ID is configured, the native exchange endpoint reports
// 501 Not Implemented before doing any work.
func TestHandleNativeExchangeNotConfigured(t *testing.T) {
	a := &Auth{} // nativeClientID == ""
	h := a.HandleNativeExchange(config.IntegratedApp{}, dbtest.New())

	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("POST", "/native", strings.NewReader(`{"id_token":"x"}`)))

	if w.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501", w.Code)
	}
}

// With a client ID configured but a malformed/empty body, the request is
// rejected with 400 before the (network-dependent) token verification.
func TestHandleNativeExchangeBadBody(t *testing.T) {
	a := &Auth{nativeClientID: "native-client"}
	h := a.HandleNativeExchange(config.IntegratedApp{}, dbtest.New())

	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("POST", "/native", strings.NewReader(`not-json`)))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
