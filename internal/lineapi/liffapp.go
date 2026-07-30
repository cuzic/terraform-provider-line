package lineapi

import (
	"fmt"
	"strings"
)

// LiffView describes a LIFF app's view configuration.
type LiffView struct {
	Type       string // one of: compact, tall, full
	URL        string
	ModuleMode *bool
}

// LiffFeatures describes optional LIFF app features.
type LiffFeatures struct {
	BLE    *bool
	QRCode *bool
}

// LiffApp is the pure domain representation of a LIFF app, independent of
// the wire format used by the LIFF server API (POST/GET/PUT/DELETE
// /liff/v1/apps).
type LiffApp struct {
	LiffID               string
	View                 LiffView
	Description          string
	Features             *LiffFeatures
	PermanentLinkPattern string
	Scope                []string
	BotPrompt            string // one of: "", normal, aggressive, none
}

var validLiffViewTypes = map[string]bool{"compact": true, "tall": true, "full": true}

var validBotPrompts = map[string]bool{"": true, "normal": true, "aggressive": true, "none": true}

var validLiffScopes = map[string]bool{"openid": true, "email": true, "profile": true, "chat_message.write": true}

// ValidateLiffApp validates a LiffApp against the constraints documented for
// AddLiffAppRequest/UpdateLiffAppRequest in line/line-openapi.
func ValidateLiffApp(app LiffApp) error {
	if app.View.URL == "" {
		return fmt.Errorf("view.url must not be empty")
	}
	if !strings.HasPrefix(app.View.URL, "https://") {
		return fmt.Errorf("view.url must use the https scheme, got %q", app.View.URL)
	}
	if strings.Contains(app.View.URL, "#") {
		return fmt.Errorf("view.url must not contain a URL fragment")
	}
	if !validLiffViewTypes[app.View.Type] {
		return fmt.Errorf("view.type must be one of compact, tall, full, got %q", app.View.Type)
	}
	if !validBotPrompts[app.BotPrompt] {
		return fmt.Errorf("bot_prompt must be one of normal, aggressive, none, got %q", app.BotPrompt)
	}
	for _, s := range app.Scope {
		if !validLiffScopes[s] {
			return fmt.Errorf("scope contains invalid value %q", s)
		}
	}
	if strings.Contains(strings.ToLower(app.Description), "line") {
		return fmt.Errorf("description must not contain %q", "LINE")
	}
	return nil
}

// FindLiffAppByID returns the app with the given ID from apps, and whether it
// was found. The LIFF server API exposes no single-item GET — only
// GetAllLIFFApps — so a Terraform resource's Read operation must fetch the
// full list and filter locally. Isolating that filter here means it can be
// unit tested without an HTTP call.
func FindLiffAppByID(apps []LiffApp, id string) (LiffApp, bool) {
	for _, a := range apps {
		if a.LiffID == id {
			return a, true
		}
	}
	return LiffApp{}, false
}
