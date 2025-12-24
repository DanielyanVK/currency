package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type APIKeyValidator interface {
	Validate(ctx context.Context, rawKey string) (exists bool, isActive bool, err error)
}

func APIKeyAuth(store APIKeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if key == "" {
				writeErr(w, http.StatusUnauthorized, errors.New("missing X-API-Key"))
				return
			}

			exists, active, err := store.Validate(r.Context(), key)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, errors.New("internal error"))
				return
			}
			if !exists {
				writeErr(w, http.StatusUnauthorized, errors.New("invalid api key"))
				return
			}
			if !active {
				writeErr(w, http.StatusForbidden, errors.New("api key is expired"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeErr(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	encError := json.NewEncoder(w).Encode(err)
	if encError != nil {
		http.Error(w, "Unknown error", http.StatusInternalServerError)
	}
}
