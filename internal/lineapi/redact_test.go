package lineapi

import "testing"

func TestRedactToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		s     string
		token string
		want  string
	}{
		{
			name:  "redacts every occurrence",
			s:     "Authorization: Bearer abc123\nretrying with abc123",
			token: "abc123",
			want:  "Authorization: Bearer REDACTED\nretrying with REDACTED",
		},
		{
			name:  "empty token is a no-op",
			s:     "Authorization: Bearer abc123",
			token: "",
			want:  "Authorization: Bearer abc123",
		},
		{
			name:  "token not present is a no-op",
			s:     "no secrets here",
			token: "abc123",
			want:  "no secrets here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RedactToken(tt.s, tt.token)
			if got != tt.want {
				t.Fatalf("RedactToken(%q, %q) = %q, want %q", tt.s, tt.token, got, tt.want)
			}
		})
	}
}
