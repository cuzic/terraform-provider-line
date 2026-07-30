package lineapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	MaxRichMenuNameLength    = 300
	MaxChatBarTextLength     = 14
	MaxRichMenuAreas         = 20
	MaxRichMenuSizeDimension = 2147483647
)

// RichMenuSize is the pixel size of a rich menu image.
type RichMenuSize struct {
	Width  int64
	Height int64
}

// RichMenuBounds is a tappable area's position and size, relative to the
// top-left corner of the rich menu image.
type RichMenuBounds struct {
	X      int64
	Y      int64
	Width  int64
	Height int64
}

// RichMenuAction is the action triggered when a rich menu area is tapped.
// LINE supports many action types (message, postback, uri, richmenuswitch,
// ...); this provider intentionally supports the three most common ones for
// its MVP.
type RichMenuAction struct {
	Type  string // one of: postback, message, uri
	Label string
	Data  string // postback only
	Text  string // message only
	URI   string // uri only
}

// RichMenuArea pairs a tappable region with the action it triggers.
type RichMenuArea struct {
	Bounds RichMenuBounds
	Action RichMenuAction
}

// RichMenu is the pure domain representation of a rich menu, independent of
// the wire format used by the Messaging API.
type RichMenu struct {
	RichMenuID  string
	Size        RichMenuSize
	Selected    bool
	Name        string
	ChatBarText string
	Areas       []RichMenuArea
}

var validRichMenuActionTypes = map[string]bool{"postback": true, "message": true, "uri": true}

// ValidateRichMenu validates a RichMenu against the constraints documented
// for RichMenuRequest in line/line-openapi, plus the geometric constraint
// (not explicit in the OpenAPI spec, but enforced by the API in practice)
// that every area must fit within the menu's overall size.
func ValidateRichMenu(rm RichMenu) error {
	if rm.Size.Width <= 0 || rm.Size.Width > MaxRichMenuSizeDimension {
		return fmt.Errorf("size.width must be between 1 and %d, got %d", MaxRichMenuSizeDimension, rm.Size.Width)
	}
	if rm.Size.Height <= 0 || rm.Size.Height > MaxRichMenuSizeDimension {
		return fmt.Errorf("size.height must be between 1 and %d, got %d", MaxRichMenuSizeDimension, rm.Size.Height)
	}
	if rm.Name == "" {
		return fmt.Errorf("name must not be empty")
	}
	if len(rm.Name) > MaxRichMenuNameLength {
		return fmt.Errorf("name must be %d characters or fewer, got %d", MaxRichMenuNameLength, len(rm.Name))
	}
	if len(rm.ChatBarText) == 0 {
		return fmt.Errorf("chat_bar_text must not be empty")
	}
	if len(rm.ChatBarText) > MaxChatBarTextLength {
		return fmt.Errorf("chat_bar_text must be %d characters or fewer, got %d", MaxChatBarTextLength, len(rm.ChatBarText))
	}
	if len(rm.Areas) == 0 {
		return fmt.Errorf("areas must contain at least one area")
	}
	if len(rm.Areas) > MaxRichMenuAreas {
		return fmt.Errorf("areas must contain %d or fewer entries, got %d", MaxRichMenuAreas, len(rm.Areas))
	}
	for i, area := range rm.Areas {
		if err := validateArea(rm.Size, area); err != nil {
			return fmt.Errorf("areas[%d]: %w", i, err)
		}
	}
	return nil
}

func validateArea(size RichMenuSize, area RichMenuArea) error {
	b := area.Bounds
	if b.Width <= 0 || b.Height <= 0 {
		return fmt.Errorf("bounds width and height must be positive, got %dx%d", b.Width, b.Height)
	}
	if b.X < 0 || b.Y < 0 {
		return fmt.Errorf("bounds x and y must be non-negative, got (%d, %d)", b.X, b.Y)
	}
	if b.X+b.Width > size.Width || b.Y+b.Height > size.Height {
		return fmt.Errorf("bounds (%d, %d, %d, %d) exceed the rich menu size (%d, %d)",
			b.X, b.Y, b.Width, b.Height, size.Width, size.Height)
	}
	return validateAction(area.Action)
}

func validateAction(action RichMenuAction) error {
	if !validRichMenuActionTypes[action.Type] {
		return fmt.Errorf("action.type must be one of postback, message, uri, got %q", action.Type)
	}
	switch action.Type {
	case "postback":
		if action.Data == "" {
			return fmt.Errorf("action.data must not be empty for a postback action")
		}
	case "message":
		if action.Text == "" {
			return fmt.Errorf("action.text must not be empty for a message action")
		}
	case "uri":
		if action.URI == "" {
			return fmt.Errorf("action.uri must not be empty for a uri action")
		}
	}
	return nil
}

// ContentHash returns a stable, content-addressed hash of a rich menu image.
//
// The Messaging API exposes no way to read back which image bytes are
// currently attached to a rich menu (GetRichMenu returns metadata only), so
// this provider stores this hash in Terraform state and compares it against
// a freshly computed hash of the configured image to decide whether the
// image needs to be re-uploaded — without it, every plan would look like a
// no-op change even after the local image file changes.
func ContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
