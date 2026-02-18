package searchindex

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/psds-microservice/operator-directory-service/internal/model"
)

// OperatorIndexer — интерфейс для индексации операторов в search-service (для подмены моком в тестах).
type OperatorIndexer interface {
	IndexOperatorAsync(profile *model.OperatorProfile)
}

// Client отправляет операторов в search-service для индексации (best-effort, не блокирует API).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient возвращает клиент. Если baseURL пустой, вызовы IndexOperator — no-op.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type indexOperatorPayload struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Region      string `json:"region"`
	Role        string `json:"role"`
}

// IndexOperator отправляет оператора в search-service.
func (c *Client) IndexOperator(ctx context.Context, profile *model.OperatorProfile) {
	if c.baseURL == "" || profile == nil {
		return
	}
	payload := indexOperatorPayload{
		UserID:      profile.UserID.String(),
		DisplayName: profile.DisplayName,
		Region:      profile.Region,
		Role:        profile.Role,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("searchindex: marshal operator: %v", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search/index/operator", bytes.NewReader(body))
	if err != nil {
		log.Printf("searchindex: new request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("searchindex: request operator: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("searchindex: status %d for operator %s", resp.StatusCode, profile.UserID)
	}
}

// IndexOperatorAsync вызывает IndexOperator в отдельной горутине.
func (c *Client) IndexOperatorAsync(profile *model.OperatorProfile) {
	if c.baseURL == "" || profile == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.IndexOperator(ctx, profile)
	}()
}
