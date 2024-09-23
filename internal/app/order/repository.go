package order

import (
	"context"
	"database/sql"
	"errors"
	"github.com/korol8484/gofermart/internal/app/domain"
)

type Repository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) LoadOrder(ctx context.Context, number string) (*domain.Order, error) {
	o := &domain.Order{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT o.id, o.number, o.status, o.user_id, o.created_at FROM orders o WHERE o.number = $1;`,
		number,
	).Scan(&o.Id, &o.Number, &o.Status, &o.UserId, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFoundOrder
		}

		return nil, err
	}

	return o, nil
}

func (r *Repository) CreateOrder(ctx context.Context, order *domain.Order) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO orders (number, status, user_id, created_at) VALUES ($1, $2, $3, $4) RETURNING id;`,
		order.Number, order.Status, order.UserId, order.CreatedAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) LoadOrdersWithBalance(ctx context.Context, userId domain.UserId) ([]domain.OrderWithBalance, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT o.id, o.number, o.status, o.user_id, o.created_at, b.sum FROM orders o LEFT JOIN balance b on o.number = b.order_number AND b.type = $1 WHERE o.user_id = $2 ORDER BY o.created_at DESC;`,
		domain.BalanceTypeAdd,
		userId,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var orders []domain.OrderWithBalance
	for rows.Next() {
		var balance sql.NullFloat64

		o := domain.OrderWithBalance{
			Order: domain.Order{},
		}

		err = rows.Scan(&o.Id, &o.Number, &o.Status, &o.UserId, &o.CreatedAt, &balance)
		if err != nil {
			return nil, err
		}

		if balance.Valid {
			o.Balance = balance.Float64
		}

		orders = append(orders, o)
	}

	return orders, nil
}
