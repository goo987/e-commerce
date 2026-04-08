package middleware

import (
	"context"
	"e-commerce/internal/repository"
	"net/http"
	"strconv"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			cookieName := "session_" + role
			cookie, err := r.Cookie(cookieName)

			if err != nil || cookie.Value == "" {
				if role == "user" {
					http.Redirect(w, r, "/register", http.StatusSeeOther)
				} else {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
				}
				return
			}

			parts := strings.Split(cookie.Value, "|")
			if len(parts) != 3 {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			userIDStr := parts[0]
			userRole := parts[2]

			if userRole != role {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_user")

		if err == nil && cookie.Value != "" {
			parts := strings.Split(cookie.Value, "|")
			if len(parts) == 3 {
				userIDStr := parts[0]
				userID, err := strconv.Atoi(userIDStr)

				if err == nil {
					ctx := context.WithValue(r.Context(), UserIDKey, userID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func TrackVisitor(repo *repository.DashboardRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				go repo.IncrementVisitor()
			}
			next.ServeHTTP(w, r)
		})
	}
}