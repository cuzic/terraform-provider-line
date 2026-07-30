package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

type richMenuSizeModel struct {
	Width  types.Int64 `tfsdk:"width"`
	Height types.Int64 `tfsdk:"height"`
}

type richMenuBoundsModel struct {
	X      types.Int64 `tfsdk:"x"`
	Y      types.Int64 `tfsdk:"y"`
	Width  types.Int64 `tfsdk:"width"`
	Height types.Int64 `tfsdk:"height"`
}

type richMenuActionModel struct {
	Type  types.String `tfsdk:"type"`
	Label types.String `tfsdk:"label"`
	Data  types.String `tfsdk:"data"`
	Text  types.String `tfsdk:"text"`
	URI   types.String `tfsdk:"uri"`
}

type richMenuAreaModel struct {
	Bounds richMenuBoundsModel `tfsdk:"bounds"`
	Action richMenuActionModel `tfsdk:"action"`
}

type richMenuResourceModel struct {
	ID          types.String        `tfsdk:"id"`
	Name        types.String        `tfsdk:"name"`
	ChatBarText types.String        `tfsdk:"chat_bar_text"`
	Selected    types.Bool          `tfsdk:"selected"`
	Size        richMenuSizeModel   `tfsdk:"size"`
	Areas       []richMenuAreaModel `tfsdk:"areas"`
	ImagePath   types.String        `tfsdk:"image_path"`
	ImageHash   types.String        `tfsdk:"image_hash"`
}

// modelToRichMenu converts the Terraform schema representation into the
// pure lineapi.RichMenu domain type. It does not touch ImagePath/ImageHash —
// those govern the separate content-upload step, not the RichMenuRequest
// body.
func modelToRichMenu(data richMenuResourceModel) lineapi.RichMenu {
	areas := make([]lineapi.RichMenuArea, 0, len(data.Areas))
	for _, a := range data.Areas {
		areas = append(areas, lineapi.RichMenuArea{
			Bounds: lineapi.RichMenuBounds{
				X:      a.Bounds.X.ValueInt64(),
				Y:      a.Bounds.Y.ValueInt64(),
				Width:  a.Bounds.Width.ValueInt64(),
				Height: a.Bounds.Height.ValueInt64(),
			},
			Action: lineapi.RichMenuAction{
				Type:  a.Action.Type.ValueString(),
				Label: a.Action.Label.ValueString(),
				Data:  a.Action.Data.ValueString(),
				Text:  a.Action.Text.ValueString(),
				URI:   a.Action.URI.ValueString(),
			},
		})
	}

	return lineapi.RichMenu{
		RichMenuID:  data.ID.ValueString(),
		Name:        data.Name.ValueString(),
		ChatBarText: data.ChatBarText.ValueString(),
		Selected:    data.Selected.ValueBool(),
		Size: lineapi.RichMenuSize{
			Width:  data.Size.Width.ValueInt64(),
			Height: data.Size.Height.ValueInt64(),
		},
		Areas: areas,
	}
}

// richMenuToModel writes a lineapi.RichMenu (as returned by GetRichMenu)
// into data, leaving ImagePath/ImageHash untouched — the API response never
// carries image content, so the caller is responsible for preserving those
// two fields from prior state/plan.
func richMenuToModel(rm lineapi.RichMenu, data *richMenuResourceModel) {
	data.ID = types.StringValue(rm.RichMenuID)
	data.Name = types.StringValue(rm.Name)
	data.ChatBarText = types.StringValue(rm.ChatBarText)
	data.Selected = types.BoolValue(rm.Selected)
	data.Size = richMenuSizeModel{
		Width:  types.Int64Value(rm.Size.Width),
		Height: types.Int64Value(rm.Size.Height),
	}

	areas := make([]richMenuAreaModel, 0, len(rm.Areas))
	for _, a := range rm.Areas {
		areas = append(areas, richMenuAreaModel{
			Bounds: richMenuBoundsModel{
				X:      types.Int64Value(a.Bounds.X),
				Y:      types.Int64Value(a.Bounds.Y),
				Width:  types.Int64Value(a.Bounds.Width),
				Height: types.Int64Value(a.Bounds.Height),
			},
			Action: richMenuActionModel{
				Type:  types.StringValue(a.Action.Type),
				Label: stringOrNull(a.Action.Label),
				Data:  stringOrNull(a.Action.Data),
				Text:  stringOrNull(a.Action.Text),
				URI:   stringOrNull(a.Action.URI),
			},
		})
	}
	data.Areas = areas
}
