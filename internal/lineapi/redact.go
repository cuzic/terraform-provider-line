package lineapi

import "strings"

// RedactToken returns s with every occurrence of token replaced by
// "REDACTED". Callers use this before writing request/response details to
// provider logs so a channel access token never ends up in a log line or
// crash log (see docs/adr/0001-channel-access-token-storage.md).
func RedactToken(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "REDACTED")
}
