package balance

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/korol8484/gofermart/internal/app/domain"
	"strings"
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
		`SELECT * FROM balance b WHERE b.user_id = $1 AND b.type = 1 ORDER BY b.created_at DESC;`,
		userId,
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
