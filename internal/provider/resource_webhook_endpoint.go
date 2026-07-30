package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cuzic/terraform-provider-line/internal/client"
	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

// singletonID is used as the Terraform resource ID for channel-level
// singleton resources — ones LINE exposes as a single GET/PUT pair with no
// list or creatable/deletable identity, such as the webhook endpoint. Any
// non-empty string import ID works, since there is only ever one instance
// per channel access token.
const webhookEndpointSingletonID = "webhook_endpoint"

func NewWebhookEndpointResource() resource.Resource {
	return &webhookEndpointResource{}
}

type webhookEndpointResource struct {
	client *client.Client
}

type webhookEndpointResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Endpoint types.String `tfsdk:"endpoint"`
	Active   types.Bool   `tfsdk:"active"`
}

var (
	_ resource.Resource                = (*webhookEndpointResource)(nil)
	_ resource.ResourceWithConfigure   = (*webhookEndpointResource)(nil)
	_ resource.ResourceWithImportState = (*webhookEndpointResource)(nil)
)

func (r *webhookEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_endpoint"
}

func (r *webhookEndpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a channel's Messaging API webhook endpoint URL (PUT/GET /v2/bot/channel/webhook/endpoint). " +
			"This is a channel-level singleton — there is exactly one per channel access token — so " +
			"there is no create/delete lifecycle in the usual sense: deleting this resource clears the " +
			"endpoint URL rather than removing anything server-side.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier for this singleton resource; not a LINE-assigned ID.",
			},
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "The webhook URL. Must use the https scheme and be at most 500 characters.",
			},
			"active": schema.BoolAttribute{
				Computed: true,
				Description: "Whether the LINE Platform will deliver webhook events to this endpoint. This is " +
					"informational only — the Messaging API exposes no operation in this provider's scope to " +
					"toggle it, so it always reflects whatever LINE currently reports.",
			},
		},
	}
}

func (r *webhookEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected resource configure type", err.Error())
		return
	}
	r.client = c
}

func (r *webhookEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data webhookEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := data.Endpoint.ValueString()
	if err := lineapi.ValidateEndpointURL(endpoint); err != nil {
		resp.Diagnostics.AddError("Invalid endpoint", err.Error())
		return
	}

	if err := r.client.SetWebhookEndpoint(ctx, endpoint); err != nil {
		resp.Diagnostics.AddError("Unable to set webhook endpoint", err.Error())
		return
	}

	r.readInto(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data webhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readInto(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data webhookEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := data.Endpoint.ValueString()
	if err := lineapi.ValidateEndpointURL(endpoint); err != nil {
		resp.Diagnostics.AddError("Invalid endpoint", err.Error())
		return
	}

	if err := r.client.SetWebhookEndpoint(ctx, endpoint); err != nil {
		resp.Diagnostics.AddError("Unable to set webhook endpoint", err.Error())
		return
	}

	r.readInto(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete clears the webhook endpoint URL. There is no delete operation for
// this resource in the Messaging API — it's a channel-level singleton — so
// this is the closest analog: setting endpoint back to the empty string,
// which SetWebhookEndpointRequest documents as an accepted value (minLength:
// 0).
func (r *webhookEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.client.SetWebhookEndpoint(ctx, ""); err != nil {
		resp.Diagnostics.AddError("Unable to clear webhook endpoint", err.Error())
	}
}

func (r *webhookEndpointResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), webhookEndpointSingletonID)...)
}

// readInto fetches the current webhook endpoint from the API and writes it
// into data, leaving data.ID set to the fixed singleton identifier.
func (r *webhookEndpointResource) readInto(ctx context.Context, data *webhookEndpointResourceModel, diags *diag.Diagnostics) {
	ep, err := r.client.GetWebhookEndpoint(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			// LINE 404s this endpoint when no webhook URL has ever been set
			// on the channel (including right after this provider's own
			// Delete, which clears it rather than truly deleting anything —
			// see Delete's doc comment). That's this singleton's empty
			// state, not a missing resource, so surface it as data instead
			// of failing every subsequent plan/refresh/destroy.
			data.ID = types.StringValue(webhookEndpointSingletonID)
			data.Endpoint = types.StringValue("")
			data.Active = types.BoolValue(false)
			return
		}
		diags.AddError("Unable to read webhook endpoint", err.Error())
		return
	}
	data.ID = types.StringValue(webhookEndpointSingletonID)
	data.Endpoint = types.StringValue(ep.Endpoint)
	data.Active = types.BoolValue(ep.Active)
}
