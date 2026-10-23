package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/nenav/nenav/backend/internal/repository"
)

type authContextKey string

const (
	UserIDKey authContextKey = "user_id"
)

// RequireAuth creates a middleware that validates the Bearer token against the Store.
func RequireAuth(store repository.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			rawToken := parts[1]
			h := sha256.Sum256([]byte(rawToken))
			tokenHash := hex.EncodeToString(h[:])

			session, err := store.GetSessionByTokenHash(r.Context(), tokenHash)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if time.Now().After(session.ExpiresAt) {
				http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
				return
			}

			if session.RevokedAt != nil {
				http.Error(w, `{"error":"session revoked"}`, http.StatusUnauthorized)
				return
			}

			// Add UserID to context
			ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin creates a middleware that ensures the user has admin privileges.
// MVP: For now it just requires authentication. Real RBAC will be added later.
func RequireAdmin(store repository.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// First apply standard auth
		authMw := RequireAuth(store)
		return authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Check if r.Context().Value(UserIDKey) has admin role
			next.ServeHTTP(w, r)
		}))
	}
}
