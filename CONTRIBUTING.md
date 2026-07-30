# Contributing

This is a young, single-maintainer project. Contributions — code, issue reports, or just usage feedback — are what keep it from becoming a bus-factor-of-one dead project, so they're genuinely welcome.

## Setup

Requires Go (see `go.mod` for the minimum version) and Terraform >= 1.0.

```sh
go build ./...
go vet ./...
go test ./...
```

## Architecture

The codebase is split into three layers, in order of "how much do you need to know about Terraform to touch this":

- `internal/lineapi` — pure domain logic: request/response shaping, validation, diffing, hashing. No HTTP, no Terraform Plugin Framework types. This is where most unit tests live, and where most bugs should be caught, because it has no I/O to mock.
- `internal/client` — the LINE API HTTP client. Thin wrappers around `internal/lineapi` types plus network calls, auth headers, retries, and error decoding.
- `internal/provider` — Terraform Plugin Framework resources. Thin adapters that convert between Terraform schema types and `internal/lineapi` types, and call `internal/client`.

When adding a new resource, prefer putting any non-trivial logic (validation, request building, response parsing, diffing) in `internal/lineapi` so it can be unit tested without a real LINE account or mocked HTTP server.

## Adding or changing a resource

1. Read the relevant section of [line/line-openapi](https://github.com/line/line-openapi) first — treat it as the source of truth for request/response shapes and which operations actually exist (many LINE resources are missing an operation you'd expect, e.g. rich menus have no update endpoint).
2. Add/extend the pure logic in `internal/lineapi` with unit tests.
3. Add/extend the HTTP client in `internal/client`.
4. Add/extend the Plugin Framework resource in `internal/provider`.
5. Add an acceptance test (`internal/provider/*_test.go`, guarded by `resource.Test` / `TF_ACC=1`) that exercises Create/Read/Update/Delete against a real LINE channel. Acceptance tests are skipped by default and require `LINE_CHANNEL_ACCESS_TOKEN` (and a test channel — see the README).
6. Update `docs/` (generated via `tfplugindocs`, do not hand-edit) and `examples/`.

## Tests

- `go test ./...` runs unit tests only (fast, no network, no credentials).
- `TF_ACC=1 LINE_CHANNEL_ACCESS_TOKEN=... go test ./... -v` additionally runs acceptance tests against a real LINE channel. Use a dedicated test channel, never a production one — acceptance tests create and delete real resources.

## Tracking upstream API changes

LINE's Messaging API evolves independently of this provider. `.github/workflows/watch-line-openapi.yml` checks weekly for new commits to the spec files this provider's client is built against and opens an issue automatically; Dependabot (`.github/dependabot.yml`) does the same for Go module and Action version updates.

## Releasing

Releases are built and signed by `.github/workflows/release.yml` via [GoReleaser](https://goreleaser.com) (`.goreleaser.yml`) whenever a `v*` tag is pushed. Before the first real release, a maintainer needs to:

1. Generate a dedicated GPG key for signing releases (not a personal key) and register its public key with the Terraform Registry when publishing the provider there.
2. Add the private key and passphrase as repository secrets `GPG_PRIVATE_KEY` and `PASSPHRASE`.
3. Tag a release (`git tag v0.1.0 && git push --tags`) and confirm the workflow produces signed archives, checksums, and `terraform-registry-manifest.json` as release assets — the Terraform Registry requires all of these to list a provider.
