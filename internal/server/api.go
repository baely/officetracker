package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"

	"github.com/baely/officetracker/pkg/model"
)

func apiRouter(service model.Service) func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/state", stateRouter(service))
		r.Route("/note", noteRouter(service))
		r.Route("/developer", developerRouter(service))
		r.Route("/report", reportRouter(service))
		r.Route("/health", healthRouter(service))
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, "not found", http.StatusNotFound)
		})
	}
}

func stateRouter(service model.Service) func(chi.Router) {
	authMethodMiddleware := AllowedAuthMethods(AuthMethodSSO, AuthMethodSecret, AuthMethodExcluded)
	return func(r chi.Router) {
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadState)).Method(http.MethodGet, "/{year}/{month}/{day}", wrap(service.GetDay))
		r.With(authMethodMiddleware, RequiredScopes(ScopeWriteState)).Method(http.MethodPut, "/{year}/{month}/{day}", wrap(service.PutDay))
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadState)).Method(http.MethodGet, "/{year}/{month}", wrap(service.GetMonth))
		r.With(authMethodMiddleware, RequiredScopes(ScopeWriteState)).Method(http.MethodPut, "/{year}/{month}", wrap(service.PutMonth))
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadState)).Method(http.MethodGet, "/{year}", wrap(service.GetYear))
	}
}

func noteRouter(service model.Service) func(chi.Router) {
	authMethodMiddleware := AllowedAuthMethods(AuthMethodSSO, AuthMethodSecret, AuthMethodExcluded)
	return func(r chi.Router) {
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadNote)).Method(http.MethodGet, "/{year}/{month}", wrap(service.GetNote))
		r.With(authMethodMiddleware, RequiredScopes(ScopeWriteNote)).Method(http.MethodPut, "/{year}/{month}", wrap(service.PutNote))
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadNote)).Method(http.MethodGet, "/{year}", wrap(service.GetNotes))
	}
}

func developerRouter(service model.Service) func(chi.Router) {
	authMethodMiddleware := AllowedAuthMethods(AuthMethodSSO)
	return func(r chi.Router) {
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadDeveloper, ScopeWriteDeveloper)).Method(http.MethodGet, "/secret", wrap(service.GetSecret))
	}
}

func reportRouter(service model.Service) func(chi.Router) {
	authMethodMiddleware := AllowedAuthMethods(AuthMethodSSO, AuthMethodExcluded)
	return func(r chi.Router) {
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadState, ScopeReadReport, ScopeWriteReport)).Method(http.MethodGet, "/pdf/{year}-attendance", wrapRaw(service.GetReport))
		r.With(authMethodMiddleware, RequiredScopes(ScopeReadState, ScopeReadReport, ScopeWriteReport)).Method(http.MethodGet, "/csv/{year}-attendance", wrapRaw(service.GetReportCSV))
	}
}

func healthRouter(service model.Service) func(chi.Router) {
	return func(r chi.Router) {
		r.Method(http.MethodGet, "/check", wrap(service.Healthcheck))
		r.With(AllowedAuthMethods(AuthMethodSecret)).Method(http.MethodGet, "/auth", wrap(service.ValidateAuth))
	}
}

func wrapRaw[T any](fn func(T) (model.Response, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := mapRequest[T](r)
		if err != nil {
			err = fmt.Errorf("failed to map request: %w", err)
			slog.Error(err.Error())
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := fn(req)
		if err != nil {
			err = fmt.Errorf("failed to execute request: %w", err)
			slog.Error(err.Error())
			writeError(w, internalErrorMsg, http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", resp.ContentType)
		_, err = w.Write(resp.Data.([]byte))
		if err != nil {
			err = fmt.Errorf("failed to write response: %w", err)
			slog.Error(err.Error())
			return
		}
	}
}

func wrap[T, U any](fn func(T) (U, error)) http.HandlerFunc {
	return wrapRaw(func(req T) (model.Response, error) {
		resp, err := fn(req)
		if err != nil {
			err = fmt.Errorf("failed to execute request: %w", err)
			return model.Response{}, err
		}

		body, err := mapResponse(resp)
		if err != nil {
			err = fmt.Errorf("failed to map response: %w", err)
			return model.Response{}, err
		}

		return model.Response{
			ContentType: "application/json",
			Data:        body,
		}, nil
	})
}

func writeError(w http.ResponseWriter, msg string, code int) {
	errMsg := model.Error{
		Code:    code,
		Message: msg,
	}
	b, err := json.Marshal(errMsg)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to marshal error: %v", err))
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	http.Error(w, string(b), code)
}

func mapRequest[T any](r *http.Request) (T, error) {
	var req T

	b, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed to read request body: %w", err)
		return *new(T), err
	}
	if len(b) > 0 {
		if err = json.Unmarshal(b, &req); err != nil {
			err = fmt.Errorf("failed to unmarshal request body: %w", err)
			return *new(T), err
		}
	}

	if err = populateUserID(&req, r); err != nil {
		err = fmt.Errorf("failed to populate user ID: %w", err)
		return *new(T), err
	}

	if err = populateUrlParams(&req, r); err != nil {
		err = fmt.Errorf("failed to populate URL params: %w", err)
		return *new(T), err
	}

	if err = populateQueryParams(&req, r); err != nil {
		err = fmt.Errorf("failed to populate query params: %w", err)
		return *new(T), err
	}

	return req, nil
}

func mapResponse[T any](resp T) ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		err = fmt.Errorf("failed to marshal response: %w", err)
		return nil, err
	}
	return b, nil
}

func getUserID(r *http.Request) (int, error) {
	userID, ok := getCtxValue(r).get(ctxUserIDKey).(int)
	if !ok {
		return 0, ErrNoUserInCtx
	}
	return userID, nil
}

func getAuthMethod(r *http.Request) (AuthMethod, error) {
	authMethod, ok := getCtxValue(r).get(ctxAuthMethodKey).(AuthMethod)
	if !ok {
		return AuthMethodUnknown, fmt.Errorf("failed to get auth method from context")
	}
	return authMethod, nil
}

func getScopes(r *http.Request) ([]Scope, error) {
	strScopes, ok := getCtxValue(r).get(ctxScopesKey).([]string)
	if !ok {
		return nil, fmt.Errorf("failed to get scopes from context")
	}
	scopes := make([]Scope, len(strScopes))
	for i, s := range strScopes {
		scopes[i] = Scope(s)
	}
	return scopes, nil
}

func populateUserID[T any](req *T, r *http.Request) error {
	v := reflect.ValueOf(req).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("meta")
		if tag == "meta" {
			meta := v.Field(i)
			metaType := meta.Type()
			for j := 0; j < metaType.NumField(); j++ {
				metaField := metaType.Field(j)
				metaFieldTag := metaField.Tag.Get("meta")
				if metaFieldTag == "user_id" {
					if meta.Field(j).CanSet() && meta.Field(j).Kind() == reflect.Int {
						userID, err := getUserID(r)
						if err != nil {
							return err
						}
						meta.Field(j).SetInt(int64(userID))
					}
				}
			}
		}
	}
	return nil
}

func populateUrlParams[T any](req *T, r *http.Request) error {
	ctx := chi.RouteContext(r.Context())
	v := reflect.ValueOf(req).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("meta")
		if tag == "meta" {
			meta := v.Field(i)
			metaType := meta.Type()
			for j := 0; j < metaType.NumField(); j++ {
				metaField := metaType.Field(j)
				metaFieldTag := metaField.Tag.Get("meta")
				if metaFieldTag != "user_id" {
					if meta.Field(j).CanSet() {
						value := ctx.URLParam(metaFieldTag)
						switch meta.Field(j).Kind() {
						case reflect.String:
							meta.Field(j).SetString(value)
						case reflect.Int:
							if x, err := strconv.Atoi(value); err == nil {
								meta.Field(j).SetInt(int64(x))
							}
						default:
							return fmt.Errorf("unsupported type: %v", meta.Field(j).Kind())
						}
					}
				}
			}
		}
	}
	return nil
}

func populateQueryParams[T any](req *T, r *http.Request) error {
	u := r.URL
	v := u.Query()
	d := schema.NewDecoder()
	d.IgnoreUnknownKeys(true)
	if err := d.Decode(req, v); err != nil {
		err = fmt.Errorf("failed to decode query params: %w", err)
		return err
	}
	return nil
}
