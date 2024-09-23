package balance

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/korol8484/gofermart/internal/app/api/util"
	"github.com/korol8484/gofermart/internal/app/domain"
	"io"
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

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type usecase interface {
	UserWithdrawals(ctx context.Context, userId domain.UserID) ([]*domain.Balance, error)
	GetUserSumWC(ctx context.Context, userId domain.UserID) (*domain.SumWC, error)
	Withdraw(ctx context.Context, userId domain.UserID, number string, sum float64) (*domain.Balance, error)
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

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	userId, ok := util.UserIdFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := &withdrawRequest{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = h.uc.Withdraw(r.Context(), userId, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, domain.ErrorNumberValidateFormat) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		} else if errors.Is(err, domain.ErrBalanceInsufficientFunds) {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RegisterRoutes(loader util.AuthSession) func(mux *chi.Mux) {
	return func(mux *chi.Mux) {
		mux.With(util.CheckAuth(loader)).Get("/api/user/withdrawals", h.list)
		mux.With(util.CheckAuth(loader)).Get("/api/user/balance", h.balance)
		mux.With(util.CheckAuth(loader)).Post("/api/user/balance/withdraw", h.withdraw)
	}
}
