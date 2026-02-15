package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type PoolOperator struct {
	UserID         string    `json:"user_id"`
	Available      bool      `json:"available"`
	ActiveSessions int       `json:"active_sessions"`
	MaxSessions    int       `json:"max_sessions"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PoolClientOpts — опции клиента operator-pool (таймаут, повторы).
type PoolClientOpts struct {
	Timeout      time.Duration // таймаут одного запроса (по умолчанию 10s)
	MaxRetries   int           // число повторов при ошибке (по умолчанию 3)
	RetryBackoff time.Duration // пауза между повторами (по умолчанию 500ms)
}

type PoolClient struct {
	baseURL    string
	httpClient *http.Client
	opts       PoolClientOpts
}

func NewPoolClient(baseURL string, opts *PoolClientOpts) *PoolClient {
	o := PoolClientOpts{
		Timeout:      10 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 500 * time.Millisecond,
	}
	if opts != nil {
		if opts.Timeout > 0 {
			o.Timeout = opts.Timeout
		}
		if opts.MaxRetries > 0 {
			o.MaxRetries = opts.MaxRetries
		}
		if opts.RetryBackoff > 0 {
			o.RetryBackoff = opts.RetryBackoff
		}
	}
	return &PoolClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: o.Timeout,
		},
		opts: o,
	}
}

func (c *PoolClient) ListOperators(ctx context.Context) ([]PoolOperator, error) {
	var lastErr error
	backoff := c.opts.RetryBackoff
	for attempt := 0; attempt <= c.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[operator-directory] operator-pool list retry %d/%d after %v", attempt, c.opts.MaxRetries, backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/operator/list", nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("operator pool request: %w", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("operator pool returned %d", resp.StatusCode)
			continue
		}
		var out struct {
			Operators []PoolOperator `json:"operators"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("decode pool response: %w", err)
			continue
		}
		resp.Body.Close()
		return out.Operators, nil
	}
	return nil, lastErr
}
