package server

import (
	"fmt"
	"log/slog"
)

type AuthMethod int

const (
	AuthMethodUnknown = AuthMethod(iota)
	AuthMethodAnonymous
	AuthMethodSSO
	AuthMethodSecret
	AuthMethodExcluded
)

type Scope string

const (
	ScopeReadState      Scope = "state:read"
	ScopeWriteState     Scope = "state:write"
	ScopeReadNote       Scope = "note:read"
	ScopeWriteNote      Scope = "note:write"
	ScopeReadDeveloper  Scope = "developer:read"
	ScopeWriteDeveloper Scope = "developer:write"
	ScopeReadReport     Scope = "report:read"
	ScopeWriteReport    Scope = "report:write"
)

func compareScopes(required, have []Scope) bool {
	lookup := make(map[Scope]struct{}, len(have))
	for _, s := range have {
		lookup[s] = struct{}{}
	}

	for _, s := range required {
		if _, ok := lookup[s]; !ok {
			slog.Info(fmt.Sprintf("missing scope: %s", s))
			return false
		}
	}

	return true
}
