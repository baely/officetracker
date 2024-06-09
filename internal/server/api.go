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

	v1 "github.com/baely/officetracker/internal/implementation/v1"
)

func apiRouter() http.Handler {
	r := chi.NewRouter().With()

	r.Handle("/state", stateRouter())
	r.Handle("/developer", developerRouter())
	r.Handle("/health", healthRouter())

	return r
}

func stateRouter() http.Handler {
	r := chi.NewRouter().With(AllowedAuthMethods(AuthMethodSSO, AuthMethodSecret, AuthMethodExcluded))

	r.Method(http.MethodGet, "/{year}/{month}/{day}", wrap(v1.GetDay))
	r.Method(http.MethodPut, "/{year}/{month}/{day}", wrap(v1.PutDay))
	r.Method(http.MethodGet, "/{year}/{month}", wrap(v1.GetMonth))
	r.Method(http.MethodPut, "/{year}/{month}", wrap(v1.PutMonth))
	r.Method(http.MethodGet, "/{year}", wrap(v1.GetYear))

	return r
}

func developerRouter() http.Handler {
	r := chi.NewRouter().With(AllowedAuthMethods(AuthMethodSSO))

	r.Method(http.MethodGet, "/secret", wrap(v1.GetSecret))

	return r
}

func healthRouter() http.Handler {
	r := chi.NewRouter()

	r.Method(http.MethodGet, "/check", wrap(v1.Healthcheck))
	r.With(AllowedAuthMethods(AuthMethodSecret)).Method(http.MethodGet, "/auth", wrap(v1.ValidateAuth))

	return r
}

func wrap[T, U any](fn func(T) (U, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := mapRequest[T](r)
		if err != nil {
			err = fmt.Errorf("failed to map request: %w", err)
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := fn(req)
		if err != nil {
			err = fmt.Errorf("failed to execute request: %w", err)
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		body, err := mapResponse(resp)
		if err != nil {
			err = fmt.Errorf("failed to map response: %w", err)
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(body)
		if err != nil {
			err = fmt.Errorf("failed to write response: %w", err)
			slog.Error(err.Error())
			return
		}
	}
}

func mapRequest[T any](r *http.Request) (T, error) {
	var req T

	b, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed to read request body: %w", err)
		return *new(T), err
	}
	if err = json.Unmarshal(b, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal request body: %w", err)
		return *new(T), err
	}

	if err = populateUserID(&req, r); err != nil {
		err = fmt.Errorf("failed to populate user ID: %w", err)
		return *new(T), err
	}

	if err = populateUrlParams(&req, r); err != nil {
		err = fmt.Errorf("failed to populate URL params: %w", err)
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
		return 0, fmt.Errorf("failed to get user ID from context")
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

func populateUserID[T any](req *T, r *http.Request) error {
	userID, err := getUserID(r)
	if err != nil {
		return err
	}
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
