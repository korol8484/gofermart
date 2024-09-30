package balance

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/korol8484/gofermart/internal/app/api/util"
	"github.com/korol8484/gofermart/internal/app/domain"
	"go.uber.org/zap"
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
	LoadWithdrawals(ctx context.Context, userID domain.UserID) ([]*domain.Balance, error)
	LoadSum(ctx context.Context, userID domain.UserID) (*domain.Sum, error)
	Withdraw(ctx context.Context, userID domain.UserID, number string, sum float64) (*domain.Balance, error)
}

type Handler struct {
	uc  usecase
	log *zap.Logger
}

func NewBalanceHandler(uc usecase, log *zap.Logger) *Handler {
	return &Handler{uc: uc, log: log}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := util.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	withdrawals, err := h.uc.LoadWithdrawals(r.Context(), userID)
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
	userID, ok := util.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sumWC, err := h.uc.LoadSum(r.Context(), userID)
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

	h.log.Info(
		"user balance",
		zap.String("response", string(b)),
	)

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := util.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
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

	b, err := h.uc.Withdraw(r.Context(), userID, req.Order, req.Sum)
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

	h.log.Info(
		"balance withdraw",
		zap.Float64("sum", b.Sum),
		zap.Int("type", int(b.Type)),
	)

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RegisterRoutes(loader util.AuthSession) func(mux *chi.Mux) {
	return func(mux *chi.Mux) {
		routes := mux.With(util.CheckAuth(loader))

		routes.Get("/api/user/withdrawals", h.list)
		routes.Get("/api/user/balance", h.balance)
		routes.Post("/api/user/balance/withdraw", h.withdraw)
	}
}
