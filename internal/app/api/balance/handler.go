package balance

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/korol8484/gofermart/internal/app/api/util"
	"github.com/korol8484/gofermart/internal/app/domain"
	"net/http"
	"time"
)

type listResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type balanceResponse struct {
	Current float64 `json:"current"`
	Sum     float64 `json:"withdrawn"`
}

type usecase interface {
	UserWithdrawals(ctx context.Context, userId domain.UserId) ([]*domain.Balance, error)
	GetUserSumWC(ctx context.Context, userId domain.UserId) (*domain.SumWC, error)
}

type Handler struct {
	uc usecase
}

func NewBalanceHandler(uc usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userId, ok := util.UserIdFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.uc.UserWithdrawals(r.Context(), userId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(withdrawals) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]*listResponse, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		resp = append(resp, &listResponse{
			Order:       withdrawal.OrderNumber,
			Sum:         withdrawal.Sum,
			ProcessedAt: withdrawal.CreatedAt.Format(time.RFC3339),
		})
	}

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) {
	userId, ok := util.UserIdFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sumWC, err := h.uc.GetUserSumWC(r.Context(), userId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := &balanceResponse{
		Current: sumWC.Current,
		Sum:     sumWC.Withdrawn,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (h *Handler) RegisterRoutes(loader util.AuthSession) func(mux *chi.Mux) {
	return func(mux *chi.Mux) {
		mux.With(util.CheckAuth(loader)).Get("/api/user/withdrawals", h.list)
		mux.With(util.CheckAuth(loader)).Get("/api/user/balance", h.balance)
	}
}
