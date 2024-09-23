package balance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/korol8484/gofermart/internal/app/domain"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserWithdrawals(ctx context.Context, userId domain.UserId) ([]*domain.Balance, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT * FROM balance b WHERE b.user_id = $1 AND b.type = $2 ORDER BY b.created_at DESC;`,
		userId,
		domain.BalanceTypeWithdrawn,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var win []*domain.Balance
	for rows.Next() {
		b := &domain.Balance{}

		err = rows.Scan(&b.Id, &b.OrderNumber, &b.Sum, &b.Type, &b.CreatedAt, &b.UserId)
		if err != nil {
			return nil, err
		}

		win = append(win, b)
	}

	return win, nil
}

func (r *Repository) Withdraw(ctx context.Context, userId domain.UserId, number string, sum float64) (*domain.Balance, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `UPDATE user_balance SET balance=balance-$1 WHERE user_id = $2;`, sum, userId)
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) {
			if e.Code == "23514" && e.ConstraintName == "user_balance_balance_check" {
				return nil, domain.ErrBalanceInsufficientFunds
			}
		}

		return nil, err
	}

	var id int64
	createdAt := time.Now()

	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO balance (order_number, sum, type, user_id, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id;`,
		number, sum, domain.BalanceTypeWithdrawn, userId, createdAt,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &domain.Balance{
		Id:          id,
		OrderNumber: number,
		UserId:      userId,
		Sum:         sum,
		Type:        domain.BalanceTypeWithdrawn,
		CreatedAt:   createdAt,
	}, nil
}

func (r *Repository) GetUserSum(ctx context.Context, userId domain.UserId, types ...domain.BalanceType) ([]*domain.SumBalance, error) {
	var (
		placeholders []string
		vals         []interface{}
	)

	vals = append(vals, userId)
	for i, v := range types {
		placeholders = append(placeholders, fmt.Sprintf("$%d",
			i+2,
		))

		vals = append(vals, v)
	}

	q := fmt.Sprintf(`SELECT b.type, b.user_id, SUM(b.sum) FROM balance b WHERE b.type IN (%s) AND user_id = $1 GROUP BY b.type, b.user_id;`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(
		ctx,
		q,
		vals...,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var s []*domain.SumBalance
	for rows.Next() {
		sb := &domain.SumBalance{}

		err = rows.Scan(&sb.Type, &sb.UserId, &sb.Sum)
		if err != nil {
			return nil, err
		}

		s = append(s, sb)
	}

	return s, nil
}
