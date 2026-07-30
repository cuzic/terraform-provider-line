package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLiffAppResource(t *testing.T) {
	testAccPreCheckWithFakeServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "line_liff_app" "this" {
  view_type  = "full"
  view_url   = "https://example.com/app"
  bot_prompt = "none"
  scope      = ["profile", "chat_message.write"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("line_liff_app.this", "id"),
					resource.TestCheckResourceAttr("line_liff_app.this", "view_type", "full"),
					resource.TestCheckResourceAttr("line_liff_app.this", "view_url", "https://example.com/app"),
					resource.TestCheckResourceAttr("line_liff_app.this", "scope.#", "2"),
				),
			},
			{
				Config: `
resource "line_liff_app" "this" {
  view_type  = "tall"
  view_url   = "https://example.com/app-v2"
  bot_prompt = "normal"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("line_liff_app.this", "view_type", "tall"),
					resource.TestCheckResourceAttr("line_liff_app.this", "view_url", "https://example.com/app-v2"),
				),
			},
			{
				// Regression test: an explicit empty string is a different
				// plan value than "omitted" (null), and a prior bug collapsed
				// both to null on read-back, which Terraform rejects as
				// "provider produced inconsistent result after apply" for a
				// value that was known (not unknown) in the plan.
				//
				// Deliberately NOT covering `scope = []` here: the client's
				// `omitempty` JSON tags mean an explicitly-empty scope is
				// indistinguishable on the wire from an omitted one, so it
				// gets LINE's non-empty default scope back — a known,
				// documented limitation (see the "scope" attribute's schema
				// description), not something this test can pass.
				Config: `
resource "line_liff_app" "this" {
  view_type   = "tall"
  view_url    = "https://example.com/app-v2"
  description = ""
}
`,
				Check: resource.TestCheckResourceAttr("line_liff_app.this", "description", ""),
			},
			{
				ResourceName:      "line_liff_app.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
