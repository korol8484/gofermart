package util

import (
	"context"
	"github.com/korol8484/gofermart/internal/app/domain"
	"net/http"
)

type AuthSession interface {
	LoadUserID(r *http.Request) (domain.UserId, error)
}

var ctxUserKey = "ctx_user_id"

func CheckAuth(loader AuthSession) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userId, err := loader.LoadUserID(r)
			if err != nil {
				http.Error(w, "", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxUserKey, userId)
		})
	}
}

func UserIdFromContext(ctx context.Context) (domain.UserId, bool) {
	userId, ok := ctx.Value(ctxUserKey).(domain.UserId)

	return userId, ok
}
