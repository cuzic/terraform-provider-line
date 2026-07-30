package provider

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// writeTestPNG writes a tiny valid PNG to a temp file and returns its path,
// so acceptance tests can exercise image_path without a committed binary
// fixture. fill distinguishes images across test steps so their content
// hashes differ.
func writeTestPNG(t *testing.T, name string, fill color.RGBA) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, fill)
		}
	}

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create %s: %v", p, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return p
}

func TestAccRichMenuResource(t *testing.T) {
	testAccPreCheckWithFakeServer(t)

	imagePathV1 := writeTestPNG(t, "menu-v1.png", color.RGBA{R: 255, A: 255})
	imagePathV2 := writeTestPNG(t, "menu-v2.png", color.RGBA{B: 255, A: 255})

	config := func(imagePath string) string {
		return fmt.Sprintf(`
resource "line_rich_menu" "this" {
  name          = "nice rich menu"
  chat_bar_text = "click"
  selected      = false
  image_path    = %q

  size = {
    width  = 2500
    height = 1686
  }

  areas = [
    {
      bounds = {
        x      = 0
        y      = 0
        width  = 2500
        height = 1686
      }
      action = {
        type  = "postback"
        data  = "action=buy&itemid=123"
        label = null
        text  = null
        uri   = null
      }
    }
  ]
}
`, imagePath)
	}

	var idAfterCreate, hashAfterCreate string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(imagePathV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("line_rich_menu.this", "id"),
					resource.TestCheckResourceAttr("line_rich_menu.this", "name", "nice rich menu"),
					resource.TestCheckResourceAttr("line_rich_menu.this", "areas.#", "1"),
					resource.TestCheckResourceAttr("line_rich_menu.this", "areas.0.action.type", "postback"),
					resource.TestCheckResourceAttrSet("line_rich_menu.this", "image_hash"),
					captureAttr("line_rich_menu.this", "id", &idAfterCreate),
					captureAttr("line_rich_menu.this", "image_hash", &hashAfterCreate),
				),
			},
			{
				// Only image_path changes: Update (not replace) should run,
				// so id must stay the same and image_hash must change.
				Config: config(imagePathV2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPtr("line_rich_menu.this", "id", &idAfterCreate),
					resource.TestCheckResourceAttrSet("line_rich_menu.this", "image_hash"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["line_rich_menu.this"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						newHash := rs.Primary.Attributes["image_hash"]
						if newHash == hashAfterCreate {
							return fmt.Errorf("expected image_hash to change after updating image_path, stayed %q", newHash)
						}
						return nil
					},
				),
			},
		},
	})
}

// captureAttr records a resource attribute's value into *out, for
// cross-step comparisons (e.g. asserting an ID is stable across an in-place
// update instead of a replace).
func captureAttr(resourceName, attr string, out *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		v, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("attribute %s not found on %s", attr, resourceName)
		}
		*out = v
		return nil
	}
}
