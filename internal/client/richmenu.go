package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

type richMenuSizeDTO struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type richMenuBoundsDTO struct {
	X      int64 `json:"x"`
	Y      int64 `json:"y"`
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

// richMenuActionDTO mirrors the LINE Action union type, restricted to the
// postback/message/uri fields this provider supports. Unused fields are
// omitted from the wire request via omitempty.
type richMenuActionDTO struct {
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
	Data  string `json:"data,omitempty"`
	Text  string `json:"text,omitempty"`
	URI   string `json:"uri,omitempty"`
}

type richMenuAreaDTO struct {
	Bounds richMenuBoundsDTO `json:"bounds"`
	Action richMenuActionDTO `json:"action"`
}

type richMenuRequestDTO struct {
	Size        richMenuSizeDTO   `json:"size"`
	Selected    bool              `json:"selected"`
	Name        string            `json:"name"`
	ChatBarText string            `json:"chatBarText"`
	Areas       []richMenuAreaDTO `json:"areas"`
}

type richMenuResponseDTO struct {
	RichMenuID  string            `json:"richMenuId"`
	Size        richMenuSizeDTO   `json:"size"`
	Selected    bool              `json:"selected"`
	Name        string            `json:"name"`
	ChatBarText string            `json:"chatBarText"`
	Areas       []richMenuAreaDTO `json:"areas"`
}

func toRichMenuAreaDTOs(areas []lineapi.RichMenuArea) []richMenuAreaDTO {
	dtos := make([]richMenuAreaDTO, 0, len(areas))
	for _, a := range areas {
		dtos = append(dtos, richMenuAreaDTO{
			Bounds: richMenuBoundsDTO{X: a.Bounds.X, Y: a.Bounds.Y, Width: a.Bounds.Width, Height: a.Bounds.Height},
			Action: richMenuActionDTO{Type: a.Action.Type, Label: a.Action.Label, Data: a.Action.Data, Text: a.Action.Text, URI: a.Action.URI},
		})
	}
	return dtos
}

func fromRichMenuAreaDTOs(dtos []richMenuAreaDTO) []lineapi.RichMenuArea {
	areas := make([]lineapi.RichMenuArea, 0, len(dtos))
	for _, d := range dtos {
		areas = append(areas, lineapi.RichMenuArea{
			Bounds: lineapi.RichMenuBounds{X: d.Bounds.X, Y: d.Bounds.Y, Width: d.Bounds.Width, Height: d.Bounds.Height},
			Action: lineapi.RichMenuAction{Type: d.Action.Type, Label: d.Action.Label, Data: d.Action.Data, Text: d.Action.Text, URI: d.Action.URI},
		})
	}
	return areas
}

func toRichMenuRequestDTO(rm lineapi.RichMenu) richMenuRequestDTO {
	return richMenuRequestDTO{
		Size:        richMenuSizeDTO{Width: rm.Size.Width, Height: rm.Size.Height},
		Selected:    rm.Selected,
		Name:        rm.Name,
		ChatBarText: rm.ChatBarText,
		Areas:       toRichMenuAreaDTOs(rm.Areas),
	}
}

func fromRichMenuResponseDTO(dto richMenuResponseDTO) lineapi.RichMenu {
	return lineapi.RichMenu{
		RichMenuID:  dto.RichMenuID,
		Size:        lineapi.RichMenuSize{Width: dto.Size.Width, Height: dto.Size.Height},
		Selected:    dto.Selected,
		Name:        dto.Name,
		ChatBarText: dto.ChatBarText,
		Areas:       fromRichMenuAreaDTOs(dto.Areas),
	}
}

type richMenuIDResponse struct {
	RichMenuID string `json:"richMenuId"`
}

// CreateRichMenu creates a rich menu (POST /v2/bot/richmenu) and returns its
// ID. There is no update endpoint for rich menu attributes — changing
// anything but the attached image requires deleting and recreating the menu.
func (c *Client) CreateRichMenu(ctx context.Context, rm lineapi.RichMenu) (string, error) {
	body, err := c.doJSON(ctx, http.MethodPost, c.apiBaseURL, "/v2/bot/richmenu", toRichMenuRequestDTO(rm))
	if err != nil {
		return "", err
	}
	var resp richMenuIDResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decode CreateRichMenu response: %w", err)
	}
	return resp.RichMenuID, nil
}

// GetRichMenu fetches a rich menu's metadata (GET /v2/bot/richmenu/{id}).
// It does not return the attached image; see UploadRichMenuImage's doc
// comment for why image state is tracked via a content hash instead.
func (c *Client) GetRichMenu(ctx context.Context, richMenuID string) (lineapi.RichMenu, error) {
	body, err := c.doJSON(ctx, http.MethodGet, c.apiBaseURL, "/v2/bot/richmenu/"+richMenuID, nil)
	if err != nil {
		return lineapi.RichMenu{}, err
	}
	var resp richMenuResponseDTO
	if err := json.Unmarshal(body, &resp); err != nil {
		return lineapi.RichMenu{}, fmt.Errorf("decode GetRichMenu response: %w", err)
	}
	return fromRichMenuResponseDTO(resp), nil
}

// DeleteRichMenu deletes a rich menu (DELETE /v2/bot/richmenu/{id}).
func (c *Client) DeleteRichMenu(ctx context.Context, richMenuID string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, c.apiBaseURL, "/v2/bot/richmenu/"+richMenuID, nil)
	return err
}

// UploadRichMenuImage attaches an image to a rich menu (POST
// /v2/bot/richmenu/{id}/content). Unlike every other endpoint this client
// calls, this one lives on api-data.line.me instead of api.line.me and takes
// a raw binary body (Content-Type: image/png or image/jpeg) rather than
// JSON — confirmed against line/line-openapi's messaging-api.yml, which
// overrides the servers block for this path specifically.
func (c *Client) UploadRichMenuImage(ctx context.Context, richMenuID, contentType string, data []byte) error {
	_, err := c.do(ctx, http.MethodPost, c.dataBaseURL, "/v2/bot/richmenu/"+richMenuID+"/content", data, contentType)
	return err
}
