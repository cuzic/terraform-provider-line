package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cuzic/terraform-provider-line/internal/client"
	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

func NewLiffAppResource() resource.Resource {
	return &liffAppResource{}
}

type liffAppResource struct {
	client *client.Client
}

type liffAppResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ViewType             types.String `tfsdk:"view_type"`
	ViewURL              types.String `tfsdk:"view_url"`
	ModuleMode           types.Bool   `tfsdk:"module_mode"`
	Description          types.String `tfsdk:"description"`
	BLE                  types.Bool   `tfsdk:"ble"`
	QRCode               types.Bool   `tfsdk:"qr_code"`
	PermanentLinkPattern types.String `tfsdk:"permanent_link_pattern"`
	Scope                types.List   `tfsdk:"scope"`
	BotPrompt            types.String `tfsdk:"bot_prompt"`
}

var (
	_ resource.Resource                = (*liffAppResource)(nil)
	_ resource.ResourceWithConfigure   = (*liffAppResource)(nil)
	_ resource.ResourceWithImportState = (*liffAppResource)(nil)
)

func (r *liffAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_liff_app"
}

func (r *liffAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LIFF app on a channel (POST/PUT/DELETE /liff/v1/apps). " +
			"There is no single-item GET in the LIFF server API, so Read fetches the full app list and " +
			"filters by ID locally.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The LIFF app ID assigned by LINE.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_type": schema.StringAttribute{
				Required:    true,
				Description: "LIFF app view size: one of compact, tall, full.",
			},
			"view_url": schema.StringAttribute{
				Required:    true,
				Description: "Endpoint URL of the web app implementing the LIFF app. Must use https and must not contain a URL fragment.",
			},
			"module_mode": schema.BoolAttribute{
				Optional:    true,
				Description: "true to run the LIFF app in modular mode (hides the header action button).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the LIFF app. Must not contain \"LINE\" or similar strings.",
			},
			"ble": schema.BoolAttribute{
				Optional:    true,
				Description: "true if the LIFF app supports Bluetooth Low Energy for LINE Things.",
			},
			"qr_code": schema.BoolAttribute{
				Optional:    true,
				Description: "true to use the 2D code reader in the LIFF app.",
			},
			"permanent_link_pattern": schema.StringAttribute{
				Optional:    true,
				Description: "How additional information in LIFF URLs is handled. LINE currently only accepts \"concat\".",
			},
			"scope": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Scopes required by the LIFF app: openid, email, profile, chat_message.write.",
			},
			"bot_prompt": schema.StringAttribute{
				Optional:    true,
				Description: "Bot link feature setting: normal, aggressive, or none (LINE's default).",
			},
		},
	}
}

func (r *liffAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected resource configure type", err.Error())
		return
	}
	r.client = c
}

func (r *liffAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data liffAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, diags := modelToLiffApp(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := lineapi.ValidateLiffApp(app); err != nil {
		resp.Diagnostics.AddError("Invalid LIFF app configuration", err.Error())
		return
	}

	liffID, err := r.client.AddLiffApp(ctx, app)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LIFF app", err.Error())
		return
	}
	data.ID = types.StringValue(liffID)

	r.readInto(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *liffAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data liffAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readInto(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.ID.IsNull() {
		// Not found: readInto leaves ID null to signal the app is gone.
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *liffAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data liffAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state liffAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	app, diags := modelToLiffApp(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := lineapi.ValidateLiffApp(app); err != nil {
		resp.Diagnostics.AddError("Invalid LIFF app configuration", err.Error())
		return
	}

	if err := r.client.UpdateLiffApp(ctx, app); err != nil {
		resp.Diagnostics.AddError("Unable to update LIFF app", err.Error())
		return
	}

	r.readInto(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *liffAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data liffAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteLiffApp(ctx, data.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LIFF app", err.Error())
	}
}

func (r *liffAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readInto fetches the full LIFF app list and writes the entry matching
// data.ID into data, since the LIFF server API has no single-item GET. If
// the app is no longer present, data.ID is set to null so callers (Read) can
// detect deletion outside Terraform.
func (r *liffAppResource) readInto(ctx context.Context, data *liffAppResourceModel, diags *diag.Diagnostics) {
	apps, err := r.client.GetAllLiffApps(ctx)
	if err != nil {
		diags.AddError("Unable to list LIFF apps", err.Error())
		return
	}

	app, ok := lineapi.FindLiffAppByID(apps, data.ID.ValueString())
	if !ok {
		data.ID = types.StringNull()
		return
	}

	liffAppToModel(ctx, app, data)
}
