package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccWebhookEndpointResource exercises Create/Update/Import against a
// fake LINE API server. It requires TF_ACC=1 (see resource.Test) and,
// against a real LINE channel, LINE_CHANNEL_ACCESS_TOKEN — but here
// testAccPreCheckWithFakeServer substitutes a local fake server, so it can
// run without either.
func TestAccWebhookEndpointResource(t *testing.T) {
	testAccPreCheckWithFakeServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "line_webhook_endpoint" "this" {
  endpoint = "https://example.com/webhook"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("line_webhook_endpoint.this", "endpoint", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("line_webhook_endpoint.this", "active", "true"),
					resource.TestCheckResourceAttr("line_webhook_endpoint.this", "id", "webhook_endpoint"),
				),
			},
			{
				Config: `
resource "line_webhook_endpoint" "this" {
  endpoint = "https://example.com/webhook-v2"
}
`,
				Check: resource.TestCheckResourceAttr("line_webhook_endpoint.this", "endpoint", "https://example.com/webhook-v2"),
			},
			{
				ResourceName:      "line_webhook_endpoint.this",
				ImportState:       true,
				ImportStateId:     "webhook_endpoint",
				ImportStateVerify: true,
			},
		},
	})
}
