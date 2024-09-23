package util

import (
	"context"
	"github.com/korol8484/gofermart/internal/app/domain"
	"net/http"
)

type AuthSession interface {
	LoadUserID(r *http.Request) (domain.UserID, error)
}

type key string

var ctxUserKey key = "ctx_user_id"

func CheckAuth(loader AuthSession) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := loader.LoadUserID(r)
			if err != nil {
				http.Error(w, "", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxUserKey, userID)

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	userID, ok := ctx.Value(ctxUserKey).(domain.UserID)

	return userID, ok
}
