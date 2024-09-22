package order

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/korol8484/gofermart/internal/app/api/util"
	"github.com/korol8484/gofermart/internal/app/domain"
	"github.com/korol8484/gofermart/internal/app/order"
	"io"
	"net/http"
	"time"
)

type listResponse struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type orderRep interface {
	CreateOrder(ctx context.Context, number string, userId domain.UserId) (*domain.Order, error)
	UserOrders(ctx context.Context, userId domain.UserId) ([]domain.OrderWithBalance, error)
}

type Handler struct {
	rep orderRep
}

func NewOrderHandler(rep orderRep) *Handler {
	return &Handler{rep: rep}
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	userId, ok := util.UserIdFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = h.rep.CreateOrder(r.Context(), string(body), userId)
	if err != nil {
		if errors.Is(err, order.ErrorIssetOrder) {
			w.WriteHeader(http.StatusOK)
			return
		} else if errors.Is(err, order.ErrorIssetOrderNotOwner) {
			w.WriteHeader(http.StatusConflict)
			return
		} else if errors.Is(err, order.ErrorInvalidFormat) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	userId, ok := util.UserIdFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := h.rep.UserOrders(r.Context(), userId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]*listResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, &listResponse{
			Number:     o.Number,
			Status:     string(o.Status),
			Accrual:    o.Balance,
			UploadedAt: o.CreatedAt.Format(time.RFC3339),
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

func (h *Handler) RegisterRoutes(loader util.AuthSession) func(mux *chi.Mux) {
	return func(mux *chi.Mux) {
		mux.With(util.CheckAuth(loader)).Post("/api/user/orders", h.createOrder)
		mux.With(util.CheckAuth(loader)).Get("/api/user/orders", h.listOrders)
	}
}
