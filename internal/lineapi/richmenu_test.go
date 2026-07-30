package lineapi

import "testing"

func validRichMenu() RichMenu {
	return RichMenu{
		Size:        RichMenuSize{Width: 2500, Height: 1686},
		Name:        "nice rich menu",
		ChatBarText: "click",
		Areas: []RichMenuArea{
			{
				Bounds: RichMenuBounds{X: 0, Y: 0, Width: 2500, Height: 1686},
				Action: RichMenuAction{Type: "postback", Data: "action=buy&itemid=123"},
			},
		},
	}
}

func TestValidateRichMenu(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(RichMenu) RichMenu
		wantErr bool
	}{
		{name: "valid menu", mutate: func(rm RichMenu) RichMenu { return rm }},
		{name: "zero width", mutate: func(rm RichMenu) RichMenu { rm.Size.Width = 0; return rm }, wantErr: true},
		{name: "zero height", mutate: func(rm RichMenu) RichMenu { rm.Size.Height = 0; return rm }, wantErr: true},
		{name: "empty name", mutate: func(rm RichMenu) RichMenu { rm.Name = ""; return rm }, wantErr: true},
		{name: "name too long", mutate: func(rm RichMenu) RichMenu {
			b := make([]byte, MaxRichMenuNameLength+1)
			rm.Name = string(b)
			return rm
		}, wantErr: true},
		{name: "empty chat bar text", mutate: func(rm RichMenu) RichMenu { rm.ChatBarText = ""; return rm }, wantErr: true},
		{name: "chat bar text too long", mutate: func(rm RichMenu) RichMenu { rm.ChatBarText = "this is way too long"; return rm }, wantErr: true},
		{name: "no areas", mutate: func(rm RichMenu) RichMenu { rm.Areas = nil; return rm }, wantErr: true},
		{name: "too many areas", mutate: func(rm RichMenu) RichMenu {
			areas := make([]RichMenuArea, MaxRichMenuAreas+1)
			for i := range areas {
				areas[i] = rm.Areas[0]
			}
			rm.Areas = areas
			return rm
		}, wantErr: true},
		{name: "area exceeds width", mutate: func(rm RichMenu) RichMenu {
			rm.Areas[0].Bounds.Width = rm.Size.Width + 1
			return rm
		}, wantErr: true},
		{name: "area exceeds height", mutate: func(rm RichMenu) RichMenu {
			rm.Areas[0].Bounds.Height = rm.Size.Height + 1
			return rm
		}, wantErr: true},
		{name: "negative x", mutate: func(rm RichMenu) RichMenu { rm.Areas[0].Bounds.X = -1; return rm }, wantErr: true},
		{name: "zero width area", mutate: func(rm RichMenu) RichMenu { rm.Areas[0].Bounds.Width = 0; return rm }, wantErr: true},
		{name: "unsupported action type", mutate: func(rm RichMenu) RichMenu { rm.Areas[0].Action = RichMenuAction{Type: "richmenuswitch"}; return rm }, wantErr: true},
		{name: "postback without data", mutate: func(rm RichMenu) RichMenu { rm.Areas[0].Action = RichMenuAction{Type: "postback"}; return rm }, wantErr: true},
		{name: "message without text", mutate: func(rm RichMenu) RichMenu { rm.Areas[0].Action = RichMenuAction{Type: "message"}; return rm }, wantErr: true},
		{name: "uri without uri", mutate: func(rm RichMenu) RichMenu { rm.Areas[0].Action = RichMenuAction{Type: "uri"}; return rm }, wantErr: true},
		{name: "valid message action", mutate: func(rm RichMenu) RichMenu {
			rm.Areas[0].Action = RichMenuAction{Type: "message", Text: "hi"}
			return rm
		}},
		{name: "valid uri action", mutate: func(rm RichMenu) RichMenu {
			rm.Areas[0].Action = RichMenuAction{Type: "uri", URI: "https://example.com"}
			return rm
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRichMenu(tt.mutate(validRichMenu()))
			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	t.Parallel()

	a := ContentHash([]byte("hello"))
	b := ContentHash([]byte("hello"))
	c := ContentHash([]byte("world"))

	if a != b {
		t.Fatalf("expected identical input to produce identical hashes, got %q and %q", a, b)
	}
	if a == c {
		t.Fatalf("expected different input to produce different hashes, both were %q", a)
	}
	if len(a) != 64 {
		t.Fatalf("expected a 64-character hex sha256 digest, got %d characters: %q", len(a), a)
	}
}
