package server

import (
	"fmt"
	"log/slog"
	"net/http"
)

type ctxValue map[string]interface{}

const (
	ctxKey           = "ctx"
	ctxUserIDKey     = "userID"
	ctxAuthMethodKey = "auth"
)

func getCtxValue(r *http.Request) ctxValue {
	ctx := r.Context()
	if v, ok := ctx.Value(ctxKey).(ctxValue); ok {
		return v
	}
	return ctxValue{}
}

func (c ctxValue) set(key string, val interface{}) ctxValue {
	slog.Info(fmt.Sprintf("setting key: %s, val: %v", key, val))
	c[key] = val
	return c
}

func (c ctxValue) get(key string) interface{} {
	return c[key]
}