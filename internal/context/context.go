package context

import (
	"net/http"
)

type CtxValue map[string]interface{}

const (
	CtxKey           = "ctx"
	CtxUserIDKey     = "userID"
	CtxAuthMethodKey = "auth"
)

func GetCtxValue(r *http.Request) CtxValue {
	ctx := r.Context()
	if v, ok := ctx.Value(CtxKey).(CtxValue); ok {
		return v
	}
	return CtxValue{}
}

func (c CtxValue) Set(key string, val interface{}) CtxValue {
	c[key] = val
	return c
}

func (c CtxValue) Get(key string) interface{} {
	return c[key]
}
