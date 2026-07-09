package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/embed"
)

// htmlCacheControl is applied to the static HTML pages so the Firebase Hosting
// CDN (which fronts Cloud Run) caches them. The pages carry no per-user data —
// everything dynamic is fetched client-side from the uncacheable /api paths — so
// a single cached copy is safe to share across every visitor. A modest max-age
// keeps deploys propagating quickly.
const htmlCacheControl = "public, max-age=600"

// Pages are composed once at startup (base + page templates, with no
// per-request data) and served thereafter as static bytes. This is what
// "ripping out SSR" means here: no template is executed per request, so the
// server returns identical, cacheable HTML to everyone.
var (
	pageIndex     = renderPage(embed.Hero)
	pageForm      = renderPage(embed.Form)
	pageSettings  = renderPage(embed.Settings)
	pageStats     = renderPage(embed.Stats)
	pageLogin     = renderPage(embed.Login)
	pageDeveloper = renderPage(embed.Developer)
	pageTos       = renderPage(embed.Tos)
	pagePrivacy   = renderPage(embed.Privacy)
	pageSuspended = renderPage(embed.Suspended)
	pageError     = renderPage(embed.Error)
)

// renderPage executes a (data-free) template once into a byte slice. It panics
// on failure so a template that still references per-request data is caught at
// startup rather than serving broken pages.
func renderPage(t *template.Template) []byte {
	var buf bytes.Buffer
	if err := t.Execute(&buf, nil); err != nil {
		panic(fmt.Sprintf("failed to render static page %q: %v", t.Name(), err))
	}
	return buf.Bytes()
}

// staticPage returns a handler that serves pre-rendered HTML with a public,
// cacheable Cache-Control header.
func staticPage(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, body)
	}
}

// servePage writes pre-rendered static HTML with a public, cacheable header.
// This deliberately overrides the "private, no-store" header that injectAuth
// sets for authenticated users — the same override the static asset handlers
// already perform — which is what allows Firebase to cache these pages even for
// logged-in visitors (the page shell holds no private data).
func servePage(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", htmlCacheControl)
	w.Write(body)
}

// serveErrorPage writes the generic error page. Error responses must not be
// cached.
func serveErrorPage(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(pageError)
}

// appContext is the uncached bootstrap payload the static frontend fetches to
// learn the viewer's auth state. The session cookie is HttpOnly so the client
// cannot read it directly; this endpoint reports only what the nav and
// page-access rules need, never any user data.
type appContext struct {
	Authenticated bool `json:"authenticated"`
	Standalone    bool `json:"standalone"`
	SSO           bool `json:"sso"`
}

// handleContext reports the viewer's auth state for the static frontend. It is
// intentionally unauthenticated (an anonymous caller simply gets all-false) and
// must never be cached, so it lives under /api/v1 where injectAuth's no-store
// applies to authenticated sessions.
func handleContext(w http.ResponseWriter, r *http.Request) {
	method, _ := getAuthMethod(r)
	ctx := appContext{
		Authenticated: method == auth.MethodSSO || method == auth.MethodSecret || method == auth.MethodExcluded,
		Standalone:    method == auth.MethodExcluded,
		SSO:           method == auth.MethodSSO,
	}
	// Anonymous integrated sessions get no header from injectAuth; ensure the
	// context response is never cached regardless of auth state.
	w.Header().Set("Cache-Control", "private, no-store")
	b, err := json.Marshal(ctx)
	if err != nil {
		writeError(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
