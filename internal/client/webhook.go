package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

type getWebhookEndpointResponse struct {
	Endpoint string `json:"endpoint"`
	Active   bool   `json:"active"`
}

type setWebhookEndpointRequest struct {
	Endpoint string `json:"endpoint"`
}

// GetWebhookEndpoint fetches the channel's current webhook endpoint
// configuration (GET /v2/bot/channel/webhook/endpoint).
func (c *Client) GetWebhookEndpoint(ctx context.Context) (lineapi.WebhookEndpoint, error) {
	body, err := c.doJSON(ctx, http.MethodGet, c.apiBaseURL, "/v2/bot/channel/webhook/endpoint", nil)
	if err != nil {
		return lineapi.WebhookEndpoint{}, err
	}
	var resp getWebhookEndpointResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return lineapi.WebhookEndpoint{}, fmt.Errorf("decode GetWebhookEndpoint response: %w", err)
	}
	return lineapi.WebhookEndpoint{Endpoint: resp.Endpoint, Active: resp.Active}, nil
}

// SetWebhookEndpoint sets the channel's webhook endpoint URL (PUT
// /v2/bot/channel/webhook/endpoint). There is no delete operation for this
// resource — it is a channel-level singleton, not a creatable/deletable
// object.
func (c *Client) SetWebhookEndpoint(ctx context.Context, endpoint string) error {
	_, err := c.doJSON(ctx, http.MethodPut, c.apiBaseURL, "/v2/bot/channel/webhook/endpoint", setWebhookEndpointRequest{Endpoint: endpoint})
	return err
}
