package accrual

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/korol8484/gofermart/internal/app/domain"
	"github.com/korol8484/gofermart/internal/app/order"
	"net/http"
	"time"
)

type response struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type Repository struct {
	client *http.Client
	host   string
}

func NewRepository(cfg *Config) *Repository {
	def := http.DefaultTransport
	def.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &Repository{
		client: &http.Client{
			Transport: def,
			Timeout:   5 * time.Second,
		},
		host: cfg.URL,
	}
}

func (r *Repository) Process(o domain.Order) (*order.AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", r.host, o.Number)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ar response
	if err = json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}

	return &order.AccrualResponse{
		Order:  ar.Order,
		Status: ar.Status,
		Sum:    ar.Accrual,
	}, nil
}
