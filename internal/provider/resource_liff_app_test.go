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
				ResourceName:      "line_liff_app.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
