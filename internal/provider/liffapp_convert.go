package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

// modelToLiffApp converts a liffAppResourceModel (Terraform schema types)
// into a lineapi.LiffApp (plain domain type), the only place in this
// resource where types.List gets unwrapped into a []string.
func modelToLiffApp(ctx context.Context, data liffAppResourceModel) (lineapi.LiffApp, diag.Diagnostics) {
	var diags diag.Diagnostics

	var scope []string
	if !data.Scope.IsNull() && !data.Scope.IsUnknown() {
		diags.Append(data.Scope.ElementsAs(ctx, &scope, false)...)
	}

	var features *lineapi.LiffFeatures
	if !data.BLE.IsNull() || !data.QRCode.IsNull() {
		features = &lineapi.LiffFeatures{}
		if !data.BLE.IsNull() {
			v := data.BLE.ValueBool()
			features.BLE = &v
		}
		if !data.QRCode.IsNull() {
			v := data.QRCode.ValueBool()
			features.QRCode = &v
		}
	}

	var moduleMode *bool
	if !data.ModuleMode.IsNull() {
		v := data.ModuleMode.ValueBool()
		moduleMode = &v
	}

	app := lineapi.LiffApp{
		LiffID: data.ID.ValueString(),
		View: lineapi.LiffView{
			Type:       data.ViewType.ValueString(),
			URL:        data.ViewURL.ValueString(),
			ModuleMode: moduleMode,
		},
		Description:          data.Description.ValueString(),
		Features:             features,
		PermanentLinkPattern: data.PermanentLinkPattern.ValueString(),
		Scope:                scope,
		BotPrompt:            data.BotPrompt.ValueString(),
	}
	return app, diags
}

// liffAppToModel writes a lineapi.LiffApp back into a liffAppResourceModel,
// the inverse of modelToLiffApp. data.ID is intentionally left untouched by
// this function; callers set it explicitly (Create sets it from the API
// response, Read/Update carry over the existing state/plan value).
func liffAppToModel(ctx context.Context, app lineapi.LiffApp, data *liffAppResourceModel) {
	data.ID = types.StringValue(app.LiffID)
	data.ViewType = types.StringValue(app.View.Type)
	data.ViewURL = types.StringValue(app.View.URL)
	data.ModuleMode = boolPtrToTF(app.View.ModuleMode)
	// These three are Optional+Computed with LINE-side defaults (see the
	// schema comment in resource_liff_app.go), so — unlike a plain Optional
	// attribute — it's safe and necessary to always write back a concrete
	// value here, even an empty string. Collapsing "" to null would make an
	// explicit `description = ""` in config fail with "provider produced
	// inconsistent result after apply", since the planned value (known,
	// non-null) would no longer match what got written to state.
	data.Description = types.StringValue(app.Description)
	data.PermanentLinkPattern = types.StringValue(app.PermanentLinkPattern)
	data.BotPrompt = types.StringValue(app.BotPrompt)

	if app.Features != nil {
		data.BLE = boolPtrToTF(app.Features.BLE)
		data.QRCode = boolPtrToTF(app.Features.QRCode)
	} else {
		data.BLE = types.BoolNull()
		data.QRCode = types.BoolNull()
	}

	// Same reasoning as above: always a concrete (possibly empty) list,
	// never null, so an explicit `scope = []` in config round-trips instead
	// of being reported back as null.
	elems := make([]types.String, 0, len(app.Scope))
	for _, s := range app.Scope {
		elems = append(elems, types.StringValue(s))
	}
	list, _ := types.ListValueFrom(ctx, types.StringType, elems)
	data.Scope = list
}

func boolPtrToTF(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
