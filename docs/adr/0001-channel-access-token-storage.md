# ADR 0001: Channel access token storage in Terraform state

## Status

Accepted, with a known limitation tracked below.

## Context

This provider authenticates to the LINE Messaging API / LIFF server API with a channel access token — a long-lived bearer credential. Terraform's execution model requires the provider configuration value to be available on every `plan`/`apply`, and Terraform persists provider configuration and all resource attributes to the state file.

Concretely, that means:

- If `channel_access_token` is set directly in a `provider "line" {}` block (e.g. from a variable), Terraform does not currently persist provider-level configuration values into the state file itself, but the value still flows through the plan file and provider logs unless care is taken.
- Every resource this provider manages is authenticated using that token, but the token itself is not stored as a resource attribute, so it does not appear in `terraform show` output for `line_*` resources.
- State file encryption at rest, remote backend access control, and log redaction are the operator's responsibility, not something Terraform or this provider can fully guarantee.

## Decision

1. **Prefer the environment variable.** The provider reads `channel_access_token` from the `LINE_CHANNEL_ACCESS_TOKEN` environment variable when the attribute is unset in configuration (see `internal/provider/provider.go`). This is the recommended way to supply the token — it keeps the value out of `.tf` files and out of version control, and out of the plan file's serialized provider config.
2. **Mark the attribute `Sensitive: true`.** This suppresses the token from Terraform CLI output (plan diffs, `terraform show`, etc.), though it does not encrypt it in the state file — Terraform's `Sensitive` marking is a display-time redaction, not encryption at rest.
3. **Never log the token.** `internal/client` takes a redaction-aware logging hook (`lineapi.RedactToken`) and the HTTP client never writes the raw `Authorization` header to logs (see `internal/client/client_test.go`'s `TestClientLogging_NeverLeaksToken`).
4. **Document the residual risk.** The README calls out that `terraform.tfstate` should be treated as sensitive (encrypted remote backend, restricted access) — this provider cannot enforce that from inside a `go build`.

## Considered alternative: short-lived tokens

LINE supports issuing short-lived channel access tokens from a `channel_id` + `channel_secret` pair (via the `channel-access-token` OpenAPI spec in `line/line-openapi`), which would reduce the blast radius of a leaked state file (a stolen short-lived token expires; a stolen long-lived one doesn't). This was **not** adopted for the MVP because:

- It would require the provider to manage token refresh across `plan`/`apply` invocations, adding meaningful complexity for comparatively little benefit at this project's current stage (single default resource set, no OIDC token management yet).
- `channel_id`/`channel_secret` are themselves credentials that would face the exact same storage question — this doesn't eliminate the problem, it moves it.

This is left as a future enhancement rather than a rejected idea; revisit if/when this provider grows a use case (e.g. CI-triggered applies) where short-lived tokens' expiry properties are worth the added complexity.

## Consequences

- Operators **must** treat their Terraform state backend as holding a live credential, exactly as they would for, say, a database password stored as a resource attribute elsewhere in their configuration.
- Recommending the environment variable over inline configuration is a convention this provider can encourage (via documentation and the `Sensitive` marking) but not enforce — a user can still write `channel_access_token = "abc123"` directly in a `.tf` file.
