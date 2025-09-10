package context

import (
	"context"
	"net/http"
)

type CtxValue map[string]interface{}

const (
	CtxKey           = "ctx"
	CtxUserIDKey     = "userID"
	CtxAuthMethodKey = "auth"
	CtxDebugKey      = "debug"
)

func MapCtx(ctx context.Context) CtxValue {
	if v, ok := ctx.Value(CtxKey).(CtxValue); ok {
		return v
	}
	return CtxValue{}
}

func GetCtxValue(r *http.Request) CtxValue {
	return MapCtx(r.Context())
}

func (c CtxValue) Set(key string, val interface{}) CtxValue {
	c[key] = val
	return c
}

func (c CtxValue) Get(key string) interface{} {
	return c[key]
}
