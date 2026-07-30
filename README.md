# terraform-provider-line

A Terraform provider for managing [LINE Messaging API](https://developers.line.biz/en/reference/messaging-api/) configuration resources declaratively — webhook endpoint, LIFF apps, and rich menus.

> **This project is not affiliated with, endorsed by, or supported by LINE Corporation or LY Corporation.** "LINE" is a trademark of LINE Corporation. This is an independent, unofficial open-source project.

## Why

LINE official account configuration (webhook URL, rich menus, LIFF apps) is normally managed by clicking through the LINE Developers console or writing one-off scripts against the Messaging API. Cloudflare has a rich official Terraform provider covering hundreds of resources; nothing comparable exists for LINE. This provider fills that gap for the subset of the API that fits Terraform's resource model.

## Scope

This provider only models **persistent configuration resources** — things with a real create/read/update/delete lifecycle. It deliberately does **not** model one-shot actions like sending or pushing messages; those belong in application code calling the Messaging API SDK directly, not in a Terraform resource.

Planned resources (see [CLAUDE.md](./CLAUDE.md) for the full rationale and roadmap):

| Resource | Status |
|---|---|
| `line_webhook_endpoint` | in progress |
| `line_liff_app` | in progress |
| `line_rich_menu` | in progress |
| `line_rich_menu_default` | planned |
| `line_rich_menu_alias` | planned |

## Usage

```hcl
provider "line" {
  # channel_access_token can also be set via the LINE_CHANNEL_ACCESS_TOKEN
  # environment variable, which is the recommended way to avoid committing
  # secrets to Terraform configuration or state.
  channel_access_token = var.line_channel_access_token
}

resource "line_webhook_endpoint" "this" {
  endpoint = "https://example.com/webhook"
}
```

## A note on secrets

The LINE channel access token you configure the provider with is a bearer credential. Terraform persists provider configuration and resource state to the state file in plaintext by default; treat your `terraform.tfstate` accordingly (remote state with encryption at rest, restricted access, etc.). See [docs/adr/0001-channel-access-token-storage.md](./docs/adr/0001-channel-access-token-storage.md) for more detail on this tradeoff.

## Development

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

[Mozilla Public License 2.0](./LICENSE).
