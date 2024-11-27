package server

import (
	"net/http"
)

type ctxValue map[string]interface{}

const (
	ctxKey           = "ctx"
	ctxUserIDKey     = "userID"
	ctxAuthMethodKey = "auth"
	ctxScopesKey     = "scopes"
)

func getCtxValue(r *http.Request) ctxValue {
	ctx := r.Context()
	if v, ok := ctx.Value(ctxKey).(ctxValue); ok {
		return v
	}
	return ctxValue{}
}

func (c ctxValue) set(key string, val interface{}) ctxValue {
	c[key] = val
	return c
}

func (c ctxValue) get(key string) interface{} {
	return c[key]
}
