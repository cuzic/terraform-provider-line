package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cuzic/terraform-provider-line/internal/client"
	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

func NewRichMenuResource() resource.Resource {
	return &richMenuResource{}
}

type richMenuResource struct {
	client *client.Client
}

var (
	_ resource.Resource                = (*richMenuResource)(nil)
	_ resource.ResourceWithConfigure   = (*richMenuResource)(nil)
	_ resource.ResourceWithImportState = (*richMenuResource)(nil)
)

func (r *richMenuResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rich_menu"
}

func (r *richMenuResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a rich menu (POST/GET/DELETE /v2/bot/richmenu) and its attached image " +
			"(POST /v2/bot/richmenu/{id}/content, on the separate api-data.line.me host). The Messaging " +
			"API has no update endpoint for a rich menu's body, so changing name, chat_bar_text, selected, " +
			"size, or areas forces replacement; only image_path can change in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Internal name for managing the rich menu; not shown to users. At most 300 characters.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"chat_bar_text": schema.StringAttribute{
				Required:    true,
				Description: "Text displayed in the chat bar. At most 14 characters.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"selected": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this is the menu shown by default. Does not by itself set it as every user's default menu — see line_rich_menu_default for that.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Pixel size of the rich menu image.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"width":  schema.Int64Attribute{Required: true},
					"height": schema.Int64Attribute{Required: true},
				},
			},
			"areas": schema.ListNestedAttribute{
				Required:    true,
				Description: "Tappable areas. At most 20.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bounds": schema.SingleNestedAttribute{
							Required: true,
							Attributes: map[string]schema.Attribute{
								"x":      schema.Int64Attribute{Required: true},
								"y":      schema.Int64Attribute{Required: true},
								"width":  schema.Int64Attribute{Required: true},
								"height": schema.Int64Attribute{Required: true},
							},
						},
						"action": schema.SingleNestedAttribute{
							Required:    true,
							Description: "Action triggered on tap. type must be one of postback, message, uri.",
							Attributes: map[string]schema.Attribute{
								"type":  schema.StringAttribute{Required: true},
								"label": schema.StringAttribute{Optional: true},
								"data":  schema.StringAttribute{Optional: true, Description: "Required when type = postback."},
								"text":  schema.StringAttribute{Optional: true, Description: "Required when type = message."},
								"uri":   schema.StringAttribute{Optional: true, Description: "Required when type = uri."},
							},
						},
					},
				},
			},
			"image_path": schema.StringAttribute{
				Optional: true,
				Description: "Local filesystem path to a .png or .jpg/.jpeg image to attach. Changing this " +
					"re-uploads the image in place without recreating the rich menu. The API has no way to " +
					"detach an image once attached, so clearing this attribute is a no-op rather than a delete.",
			},
			"image_hash": schema.StringAttribute{
				Computed: true,
				Description: "SHA-256 hash of the last successfully uploaded image, used to detect drift in " +
					"image_path's content since GetRichMenu never returns the image itself.",
			},
		},
	}
}

func (r *richMenuResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected resource configure type", err.Error())
		return
	}
	r.client = c
}

func (r *richMenuResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data richMenuResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rm := modelToRichMenu(data)
	if err := lineapi.ValidateRichMenu(rm); err != nil {
		resp.Diagnostics.AddError("Invalid rich menu configuration", err.Error())
		return
	}

	id, err := r.client.CreateRichMenu(ctx, rm)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create rich menu", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	data.ImageHash = types.StringNull()
	if !data.ImagePath.IsNull() && data.ImagePath.ValueString() != "" {
		hash, err := r.uploadImage(ctx, id, data.ImagePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to upload rich menu image", err.Error())
			return
		}
		data.ImageHash = types.StringValue(hash)
	}

	r.refresh(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *richMenuResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data richMenuResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.refresh(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update only ever runs for a change to image_path/image_hash: every other
// attribute carries a RequiresReplace plan modifier, so Terraform handles
// those changes as delete+create instead of calling this method.
func (r *richMenuResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan richMenuResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state richMenuResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	if !plan.ImagePath.IsNull() && plan.ImagePath.ValueString() != "" {
		hash, err := r.uploadImage(ctx, plan.ID.ValueString(), plan.ImagePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to upload rich menu image", err.Error())
			return
		}
		plan.ImageHash = types.StringValue(hash)
	} else {
		// The API exposes no way to detach an image once uploaded; keep
		// whatever was previously recorded rather than pretending it's gone.
		resp.Diagnostics.AddWarning(
			"image_path removed but no unattach operation exists",
			"The Messaging API has no operation to remove a rich menu's image, so the previously uploaded image remains attached. image_hash is left as-is.",
		)
		plan.ImageHash = state.ImageHash
	}

	r.refresh(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *richMenuResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data richMenuResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRichMenu(ctx, data.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete rich menu", err.Error())
	}
}

func (r *richMenuResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	// image_path/image_hash are left unset on import: the API exposes no way
	// to read back the currently attached image, so there's nothing to
	// import them from. The next plan will show image_hash going from null
	// to a value once image_path is set in configuration.
}

func (r *richMenuResource) uploadImage(ctx context.Context, richMenuID, imagePath string) (string, error) {
	contentType, err := lineapi.DetectImageContentType(imagePath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	if err := r.client.UploadRichMenuImage(ctx, richMenuID, contentType, data); err != nil {
		return "", err
	}
	return lineapi.ContentHash(data), nil
}

// refresh re-fetches the rich menu's metadata from the API and writes it
// into data, preserving ImagePath/ImageHash (never returned by the API) from
// whatever data already held. If the rich menu no longer exists, data.ID is
// set to null.
func (r *richMenuResource) refresh(ctx context.Context, data *richMenuResourceModel, diags *diag.Diagnostics) {
	imagePath, imageHash := data.ImagePath, data.ImageHash

	rm, err := r.client.GetRichMenu(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Unable to read rich menu", err.Error())
		return
	}

	richMenuToModel(rm, data)
	data.ImagePath = imagePath
	data.ImageHash = imageHash
}
