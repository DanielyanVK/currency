package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"service-currency/internal/models"
	"strings"
)

type APIKeyStore interface {
	Validate(ctx context.Context, rawKey string) (exists bool, isActive bool, err error)
}

func APIKeyAuth(store APIKeyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if key == "" {
				writeBizErr(w, http.StatusUnauthorized, "api_key_missing", "missing X-API-Key")
				return
			}

			exists, active, err := store.Validate(r.Context(), key)
			if err != nil {
				writeBizErr(w, http.StatusInternalServerError, "internal_error", "internal error")
				return
			}
			if !exists {
				writeBizErr(w, http.StatusUnauthorized, "invalid_api_key", "invalid api key")
				return
			}
			if !active {
				writeBizErr(w, http.StatusForbidden, "api_key_expired", "api key is expired")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeBizErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.BusinessError{Code: code, Message: msg})
}
