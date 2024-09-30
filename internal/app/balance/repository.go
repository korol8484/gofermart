package balance

import (
	"context"
	"database/sql"
	"fmt"
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

func (r *Repository) GetUserWithdrawals(ctx context.Context, userID domain.UserID) ([]*domain.Balance, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT * FROM balance b WHERE b.user_id = $1 AND b.type = $2 ORDER BY b.created_at DESC;`,
		userID,
		domain.BalanceTypeWithdrawn,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var win []*domain.Balance
	for rows.Next() {
		b := &domain.Balance{}
		var sum int64

		err = rows.Scan(&b.ID, &b.OrderNumber, &sum, &b.Type, &b.CreatedAt, &b.UserID)
		if err != nil {
			return nil, err
		}

		b.Sum = domain.ConvertFromCurrencyUnit(sum)
		win = append(win, b)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return win, nil
}

func (r *Repository) Withdraw(ctx context.Context, userID domain.UserID, number string, sum float64) (*domain.Balance, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var currentBalance sql.NullInt64
	err = tx.QueryRowContext(
		ctx,
		`SELECT sum(b.sum) - (
			SELECT sum(b2.sum) FROM balance b2 WHERE b2.user_id = $1 and b2.type = $3
			) as sum FROM balance b
		WHERE b.user_id = $1 and b.type = $2;`,
		userID, domain.BalanceTypeAdd, domain.BalanceTypeWithdrawn,
	).Scan(&currentBalance)
	if err != nil {
		return nil, err
	}

	if currentBalance.Valid && (currentBalance.Int64-domain.ConvertToCurrencyUnit(sum)) < 0 {
		return nil, domain.ErrBalanceInsufficientFunds
	}

	var id int64
	createdAt := time.Now()

	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO balance (order_number, sum, type, user_id, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id;`,
		number, domain.ConvertToCurrencyUnit(sum), domain.BalanceTypeWithdrawn, userID, createdAt,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &domain.Balance{
		ID:          id,
		OrderNumber: number,
		UserID:      userID,
		Sum:         sum,
		Type:        domain.BalanceTypeWithdrawn,
		CreatedAt:   createdAt,
	}, nil
}

func (r *Repository) AddBalance(o *domain.Balance) error {
	_, err := r.db.Exec(
		`INSERT INTO balance (order_number, sum, type, user_id, created_at) VALUES ($1, $2, $3, $4, $5);`,
		o.OrderNumber, domain.ConvertToCurrencyUnit(o.Sum), o.Type, o.UserID, o.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetUserSum(ctx context.Context, userID domain.UserID, types ...domain.BalanceType) ([]*domain.SumBalance, error) {
	var (
		placeholders []string
		vals         []interface{}
	)

	vals = append(vals, userID)
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

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	defer rows.Close()

	var s []*domain.SumBalance
	for rows.Next() {
		sb := &domain.SumBalance{}
		var sum int64

		err = rows.Scan(&sb.Type, &sb.UserID, &sum)
		if err != nil {
			return nil, err
		}

		sb.Sum = domain.ConvertFromCurrencyUnit(sum)
		s = append(s, sb)
	}

	return s, nil
}
