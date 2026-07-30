package lineapi

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateEndpointURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid https url", input: "https://example.com/webhook"},
		{name: "empty", input: "", wantErr: ErrEndpointEmpty},
		{name: "http scheme rejected", input: "http://example.com/webhook", wantErr: ErrEndpointNotHTTPS},
		{name: "no scheme rejected", input: "example.com/webhook", wantErr: ErrEndpointNotHTTPS},
		{name: "too long", input: "https://example.com/" + strings.Repeat("a", 500), wantErr: ErrEndpointTooLong},
		{name: "exactly max length is ok", input: "https://example.com/" + strings.Repeat("a", 500-len("https://example.com/"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEndpointURL(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ValidateEndpointURL(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateEndpointURL(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestValidateEndpointURLRejectsUnparsableURL(t *testing.T) {
	t.Parallel()

	// A percent-sign followed by non-hex digits is invalid percent-encoding
	// and makes url.Parse fail, independent of the scheme check.
	err := ValidateEndpointURL("https://example.com/%zz")
	if err == nil {
		t.Fatal("expected an error for an unparsable URL, got nil")
	}
}
