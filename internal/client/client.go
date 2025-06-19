package client

import (
	"context"
	"encoding/json"
	"net/http"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	httpClient HTTPClient
}

func NewClient(baseURL string, client HTTPClient) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (c *Client) GetOrder(ctx context.Context, orderNumber string) (*OrderResponse, error) {
	url := c.baseURL + "/api/orders/" + orderNumber
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, HandleErrorResponse(resp)
	}

	var result OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func HandleErrorResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return NewRateLimitError(resp.Header)
	case http.StatusNoContent:
		return ErrOrderNotRegistered
	default:
		return ErrServiceUnavailable
	}
}
