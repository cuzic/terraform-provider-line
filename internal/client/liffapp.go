package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

type liffViewDTO struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	ModuleMode *bool  `json:"moduleMode,omitempty"`
}

type liffFeaturesDTO struct {
	BLE    *bool `json:"ble,omitempty"`
	QRCode *bool `json:"qrCode,omitempty"`
}

type liffAppDTO struct {
	LiffID               string           `json:"liffId"`
	View                 liffViewDTO      `json:"view"`
	Description          string           `json:"description,omitempty"`
	Features             *liffFeaturesDTO `json:"features,omitempty"`
	PermanentLinkPattern string           `json:"permanentLinkPattern,omitempty"`
	Scope                []string         `json:"scope,omitempty"`
	BotPrompt            string           `json:"botPrompt,omitempty"`
}

// updateLiffAppDTO mirrors UpdateLiffAppRequest, which — unlike
// AddLiffAppRequest — has no liffId field (the ID is in the path).
type updateLiffAppDTO struct {
	View                 liffViewDTO      `json:"view"`
	Description          string           `json:"description,omitempty"`
	Features             *liffFeaturesDTO `json:"features,omitempty"`
	PermanentLinkPattern string           `json:"permanentLinkPattern,omitempty"`
	Scope                []string         `json:"scope,omitempty"`
	BotPrompt            string           `json:"botPrompt,omitempty"`
}

func toLiffFeaturesDTO(f *lineapi.LiffFeatures) *liffFeaturesDTO {
	if f == nil {
		return nil
	}
	return &liffFeaturesDTO{BLE: f.BLE, QRCode: f.QRCode}
}

func fromLiffFeaturesDTO(f *liffFeaturesDTO) *lineapi.LiffFeatures {
	if f == nil {
		return nil
	}
	return &lineapi.LiffFeatures{BLE: f.BLE, QRCode: f.QRCode}
}

func toLiffAppDTO(app lineapi.LiffApp) liffAppDTO {
	return liffAppDTO{
		LiffID:               app.LiffID,
		View:                 liffViewDTO{Type: app.View.Type, URL: app.View.URL, ModuleMode: app.View.ModuleMode},
		Description:          app.Description,
		Features:             toLiffFeaturesDTO(app.Features),
		PermanentLinkPattern: app.PermanentLinkPattern,
		Scope:                app.Scope,
		BotPrompt:            app.BotPrompt,
	}
}

func toUpdateLiffAppDTO(app lineapi.LiffApp) updateLiffAppDTO {
	return updateLiffAppDTO{
		View:                 liffViewDTO{Type: app.View.Type, URL: app.View.URL, ModuleMode: app.View.ModuleMode},
		Description:          app.Description,
		Features:             toLiffFeaturesDTO(app.Features),
		PermanentLinkPattern: app.PermanentLinkPattern,
		Scope:                app.Scope,
		BotPrompt:            app.BotPrompt,
	}
}

func fromLiffAppDTO(dto liffAppDTO) lineapi.LiffApp {
	return lineapi.LiffApp{
		LiffID:               dto.LiffID,
		View:                 lineapi.LiffView{Type: dto.View.Type, URL: dto.View.URL, ModuleMode: dto.View.ModuleMode},
		Description:          dto.Description,
		Features:             fromLiffFeaturesDTO(dto.Features),
		PermanentLinkPattern: dto.PermanentLinkPattern,
		Scope:                dto.Scope,
		BotPrompt:            dto.BotPrompt,
	}
}

type addLiffAppResponse struct {
	LiffID string `json:"liffId"`
}

// AddLiffApp creates a new LIFF app (POST /liff/v1/apps) and returns its ID.
func (c *Client) AddLiffApp(ctx context.Context, app lineapi.LiffApp) (string, error) {
	body, err := c.doJSON(ctx, http.MethodPost, c.apiBaseURL, "/liff/v1/apps", toLiffAppDTO(app))
	if err != nil {
		return "", err
	}
	var resp addLiffAppResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decode AddLiffApp response: %w", err)
	}
	return resp.LiffID, nil
}

type getAllLiffAppsResponse struct {
	Apps []liffAppDTO `json:"apps"`
}

// GetAllLiffApps lists every LIFF app on the channel (GET /liff/v1/apps).
// There is no single-item GET in the LIFF server API; callers that need one
// app should fetch this list and use lineapi.FindLiffAppByID.
func (c *Client) GetAllLiffApps(ctx context.Context) ([]lineapi.LiffApp, error) {
	body, err := c.doJSON(ctx, http.MethodGet, c.apiBaseURL, "/liff/v1/apps", nil)
	if err != nil {
		return nil, err
	}
	var resp getAllLiffAppsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode GetAllLiffApps response: %w", err)
	}
	apps := make([]lineapi.LiffApp, 0, len(resp.Apps))
	for _, dto := range resp.Apps {
		apps = append(apps, fromLiffAppDTO(dto))
	}
	return apps, nil
}

// UpdateLiffApp updates an existing LIFF app (PUT /liff/v1/apps/{liffId}).
func (c *Client) UpdateLiffApp(ctx context.Context, app lineapi.LiffApp) error {
	_, err := c.doJSON(ctx, http.MethodPut, c.apiBaseURL, "/liff/v1/apps/"+app.LiffID, toUpdateLiffAppDTO(app))
	return err
}

// DeleteLiffApp deletes a LIFF app (DELETE /liff/v1/apps/{liffId}).
func (c *Client) DeleteLiffApp(ctx context.Context, liffID string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, c.apiBaseURL, "/liff/v1/apps/"+liffID, nil)
	return err
}
