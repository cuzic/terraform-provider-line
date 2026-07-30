// Package lineapi holds the pure domain logic shared by the LINE API client
// and the Terraform provider resources: validation, request/response shaping,
// diffing, and hashing. Nothing in this package performs I/O, so all of it is
// unit-testable without a mocked HTTP server or a real LINE account.
package lineapi

import (
	"errors"
	"fmt"
	"net/url"
)

// MaxWebhookEndpointLength is the maximum length LINE accepts for a webhook
// endpoint URL, per SetWebhookEndpointRequest in line/line-openapi.
const MaxWebhookEndpointLength = 500

// WebhookEndpoint is the pure domain representation of a channel's webhook
// endpoint configuration.
type WebhookEndpoint struct {
	Endpoint string
	Active   bool
}

var (
	ErrEndpointEmpty    = errors.New("endpoint must not be empty")
	ErrEndpointTooLong  = fmt.Errorf("endpoint must be %d characters or fewer", MaxWebhookEndpointLength)
	ErrEndpointNotHTTPS = errors.New("endpoint must use the https scheme")
)

// ValidateEndpointURL validates a webhook endpoint URL against the
// constraints documented for SetWebhookEndpointRequest: non-empty, at most
// MaxWebhookEndpointLength characters, and https.
func ValidateEndpointURL(raw string) error {
	if raw == "" {
		return ErrEndpointEmpty
	}
	if len(raw) > MaxWebhookEndpointLength {
		return ErrEndpointTooLong
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("endpoint is not a valid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return ErrEndpointNotHTTPS
	}
	return nil
}
