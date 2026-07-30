package lineapi

import "testing"

func validLiffApp() LiffApp {
	return LiffApp{
		LiffID: "1234567890-abcdefgh",
		View: LiffView{
			Type: "full",
			URL:  "https://example.com/app",
		},
		BotPrompt: "none",
	}
}

func TestValidateLiffApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(LiffApp) LiffApp
		wantErr bool
	}{
		{name: "valid app", mutate: func(a LiffApp) LiffApp { return a }},
		{name: "empty url", mutate: func(a LiffApp) LiffApp { a.View.URL = ""; return a }, wantErr: true},
		{name: "non-https url", mutate: func(a LiffApp) LiffApp { a.View.URL = "http://example.com"; return a }, wantErr: true},
		{name: "url with fragment", mutate: func(a LiffApp) LiffApp { a.View.URL = "https://example.com#frag"; return a }, wantErr: true},
		{name: "invalid view type", mutate: func(a LiffApp) LiffApp { a.View.Type = "huge"; return a }, wantErr: true},
		{name: "invalid bot prompt", mutate: func(a LiffApp) LiffApp { a.BotPrompt = "loud"; return a }, wantErr: true},
		{name: "invalid scope", mutate: func(a LiffApp) LiffApp { a.Scope = []string{"admin"}; return a }, wantErr: true},
		{name: "valid scope", mutate: func(a LiffApp) LiffApp { a.Scope = []string{"profile", "chat_message.write"}; return a }},
		{name: "description contains LINE", mutate: func(a LiffApp) LiffApp { a.Description = "My LINE App"; return a }, wantErr: true},
		{name: "description case-insensitive", mutate: func(a LiffApp) LiffApp { a.Description = "my Line app"; return a }, wantErr: true},
		{name: "description containing LINE as a substring is allowed", mutate: func(a LiffApp) LiffApp { a.Description = "Online Store"; return a }},
		{name: "Timeline is allowed", mutate: func(a LiffApp) LiffApp { a.Description = "My Timeline Viewer"; return a }},
		{name: "Guidelines is allowed", mutate: func(a LiffApp) LiffApp { a.Description = "Community Guidelines"; return a }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateLiffApp(tt.mutate(validLiffApp()))
			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindLiffAppByID(t *testing.T) {
	t.Parallel()

	apps := []LiffApp{
		{LiffID: "aaa"},
		{LiffID: "bbb", Description: "target"},
		{LiffID: "ccc"},
	}

	got, ok := FindLiffAppByID(apps, "bbb")
	if !ok {
		t.Fatal("expected to find app bbb")
	}
	if got.Description != "target" {
		t.Fatalf("got wrong app: %+v", got)
	}

	_, ok = FindLiffAppByID(apps, "missing")
	if ok {
		t.Fatal("expected not to find app with id \"missing\"")
	}

	_, ok = FindLiffAppByID(nil, "aaa")
	if ok {
		t.Fatal("expected not to find anything in a nil slice")
	}
}
