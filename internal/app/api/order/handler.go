package order

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/korol8484/gofermart/internal/app/api/util"
	"github.com/korol8484/gofermart/internal/app/domain"
	"net/http"
)

type ValidateError error

var (
	ErrorIssetOrder         ValidateError = errors.New("номер заказа уже был загружен этим пользователем")
	ErrorIssetOrderNotOwner ValidateError = errors.New("номер заказа уже был загружен другим пользователем")
	ErrorInvalidFormat      ValidateError = errors.New("неверный формат номера заказа")
)

type ValidateOrder interface {
	Validate(number string) (bool, ValidateError)
}

type OrdersRepository interface {
	CreateOrder(order *domain.Order) error
	LoadOrdersWithBalance(userId domain.UserId) ([]domain.OrderWithBalance, error)
}

type Handler struct {
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RegisterRoutes(loader util.AuthSession) func(mux *chi.Mux) {
	return func(mux *chi.Mux) {
		mux.With(util.CheckAuth(loader)).Post("/api/user/orders", h.createOrder)
		mux.With(util.CheckAuth(loader)).Get("/api/user/orders", h.listOrders)
	}
}
