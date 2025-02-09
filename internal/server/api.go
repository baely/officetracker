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

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/context"
	v1 "github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/pkg/model"
)

func apiRouter(service *v1.Service) func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/state", stateRouter(service))
		r.Route("/note", noteRouter(service))
		r.Route("/settings", settingsRouter(service))
		r.Route("/developer", developerRouter(service))
		r.Route("/report", reportRouter(service))
		r.Route("/health", healthRouter(service))
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, "not found", http.StatusNotFound)
		})
	}
}

func stateRouter(service *v1.Service) func(chi.Router) {
	middlewares := chi.Middlewares{AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret, auth.MethodExcluded)}
	return func(r chi.Router) {
		r.With(middlewares...).Method(http.MethodGet, "/{year}/{month}/{day}", wrap(service.GetDay))
		r.With(middlewares...).Method(http.MethodPut, "/{year}/{month}/{day}", wrap(service.PutDay))
		r.With(middlewares...).Method(http.MethodGet, "/{year}/{month}", wrap(service.GetMonth))
		r.With(middlewares...).Method(http.MethodPut, "/{year}/{month}", wrap(service.PutMonth))
		r.With(middlewares...).Method(http.MethodGet, "/{year}", wrap(service.GetYear))
	}
}

func noteRouter(service *v1.Service) func(chi.Router) {
	middlewares := chi.Middlewares{AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret, auth.MethodExcluded)}
	return func(r chi.Router) {
		r.With(middlewares...).Method(http.MethodGet, "/{year}/{month}", wrap(service.GetNote))
		r.With(middlewares...).Method(http.MethodPut, "/{year}/{month}", wrap(service.PutNote))
		r.With(middlewares...).Method(http.MethodGet, "/{year}", wrap(service.GetNotes))
	}
}

func settingsRouter(service *v1.Service) func(router chi.Router) {
	middlewares := []func(handler http.Handler) http.Handler{AllowedAuthMethods(auth.MethodSSO)}
	return func(r chi.Router) {
		r.With(middlewares...).Method(http.MethodGet, "/", wrap(service.GetSettings))
	}
}

func developerRouter(service *v1.Service) func(chi.Router) {
	middlewares := chi.Middlewares{AllowedAuthMethods(auth.MethodSSO)}
	return func(r chi.Router) {
		r.With(middlewares...).Method(http.MethodGet, "/secret", wrap(service.GetSecret))
	}
}

func reportRouter(service *v1.Service) func(chi.Router) {
	middlewares := chi.Middlewares{AllowedAuthMethods(auth.MethodSSO, auth.MethodExcluded)}
	return func(r chi.Router) {
		r.With(middlewares...).Method(http.MethodGet, "/pdf/{year}-attendance", wrapRaw(service.GetReport))
		r.With(middlewares...).Method(http.MethodGet, "/csv/{year}-attendance", wrapRaw(service.GetReportCSV))
	}
}

func healthRouter(service *v1.Service) func(chi.Router) {
	return func(r chi.Router) {
		r.Method(http.MethodGet, "/check", wrap(service.Healthcheck))
		r.With(AllowedAuthMethods(auth.MethodSecret)).Method(http.MethodGet, "/auth", wrap(service.ValidateAuth))
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
	userID, ok := context.GetCtxValue(r).Get(context.CtxUserIDKey).(int)
	if !ok {
		return 0, ErrNoUserInCtx
	}
	return userID, nil
}

func getAuthMethod(r *http.Request) (auth.Method, error) {
	authMethod, ok := context.GetCtxValue(r).Get(context.CtxAuthMethodKey).(auth.Method)
	if !ok {
		return auth.MethodUnknown, fmt.Errorf("failed to get auth method from context")
	}
	return authMethod, nil
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
