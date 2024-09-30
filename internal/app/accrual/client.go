package accrual

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/korol8484/gofermart/internal/app/domain"
	"github.com/korol8484/gofermart/internal/app/order"
	"net/http"
	"strconv"
	"time"
)

type response struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type Client struct {
	client *http.Client
	host   string
}

func NewClient(cfg *Config) *Client {
	def := http.DefaultTransport
	def.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &Client{
		client: &http.Client{
			Transport: def,
			Timeout:   5 * time.Second,
		},
		host: cfg.URL,
	}
}

func (r *Client) Process(o domain.Order) (*order.AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", r.host, o.Number)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		errResp := &order.ErrAccrualRetry{}
		if s, ok := resp.Header["Retry-After"]; ok {
			if sleep, err := strconv.ParseInt(s[0], 10, 32); err == nil {
				errResp.WithRetryTime(sleep)
			}
		}

		return nil, errResp
	}

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
