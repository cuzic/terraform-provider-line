// Package provider implements the Terraform Plugin Framework provider for
// LINE Messaging API configuration resources. Resources in this package are
// thin adapters: they convert between Terraform schema types and
// internal/lineapi domain types, and delegate to internal/client for I/O.
// Validation and other non-trivial logic belongs in internal/lineapi, not
// here.
package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cuzic/terraform-provider-line/internal/client"
)

// channelAccessTokenEnvVar is the environment variable fallback for the
// provider's channel_access_token attribute, following the usual Terraform
// provider convention of allowing secrets to come from the environment
// instead of configuration/state.
const channelAccessTokenEnvVar = "LINE_CHANNEL_ACCESS_TOKEN"

// New returns a provider.Provider factory, parameterized by the build
// version reported by main.go (via ldflags), for registration with the
// Terraform Plugin Framework server.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &lineProvider{version: version}
	}
}

type lineProvider struct {
	version string
}

type lineProviderModel struct {
	ChannelAccessToken types.String `tfsdk:"channel_access_token"`
}

var _ provider.Provider = (*lineProvider)(nil)

func (p *lineProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "line"
	resp.Version = p.version
}

func (p *lineProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages LINE Messaging API configuration resources (webhook endpoint, LIFF apps, rich menus) declaratively. Unofficial and unaffiliated with LINE Corporation / LY Corporation.",
		Attributes: map[string]schema.Attribute{
			"channel_access_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: fmt.Sprintf(
					"LINE channel access token used to authenticate to the Messaging API and LIFF server API. "+
						"Can also be set via the %s environment variable, which is recommended so the token isn't "+
						"written into Terraform configuration (it will still be stored in state; see the README's "+
						"note on secrets).",
					channelAccessTokenEnvVar,
				),
			},
		},
	}
}

func (p *lineProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data lineProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ChannelAccessToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("channel_access_token"),
			"Unknown LINE channel access token",
			"The provider cannot be configured because channel_access_token is unknown at plan time. "+
				"This usually means it depends on a resource that has not been created yet; set it via the "+
				channelAccessTokenEnvVar+" environment variable instead if that's not intentional.",
		)
		return
	}

	token := data.ChannelAccessToken.ValueString()
	if token == "" {
		token = os.Getenv(channelAccessTokenEnvVar)
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing LINE channel access token",
			"Set channel_access_token in the provider configuration block, or the "+channelAccessTokenEnvVar+" environment variable.",
		)
		return
	}

	opts := []client.Option{
		client.WithUserAgent("terraform-provider-line/" + p.version),
		client.WithLogger(func(ctx context.Context, format string, args ...any) {
			tflog.Debug(ctx, fmt.Sprintf(format, args...))
		}),
	}
	// LINE_API_BASE_URL / LINE_DATA_API_BASE_URL let tests point the client
	// at a local fake server instead of the real LINE API; they are
	// intentionally not provider schema attributes. Only honored when they
	// point at a loopback address (which is all this provider's own test
	// suite ever sets them to) — otherwise anything able to set an env var
	// in the Terraform process could silently redirect all API traffic,
	// including the bearer token, to an arbitrary host.
	if u := os.Getenv("LINE_API_BASE_URL"); u != "" {
		if isLoopbackBaseURL(u) {
			opts = append(opts, client.WithAPIBaseURL(u))
		} else {
			resp.Diagnostics.AddWarning(
				"Ignoring LINE_API_BASE_URL",
				"LINE_API_BASE_URL is only honored when it points at a loopback address (127.0.0.1/::1/localhost); the real LINE API will be used instead.",
			)
		}
	}
	if u := os.Getenv("LINE_DATA_API_BASE_URL"); u != "" {
		if isLoopbackBaseURL(u) {
			opts = append(opts, client.WithDataBaseURL(u))
		} else {
			resp.Diagnostics.AddWarning(
				"Ignoring LINE_DATA_API_BASE_URL",
				"LINE_DATA_API_BASE_URL is only honored when it points at a loopback address (127.0.0.1/::1/localhost); the real LINE API will be used instead.",
			)
		}
	}

	c := client.New(token, opts...)

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *lineProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWebhookEndpointResource,
		NewLiffAppResource,
		NewRichMenuResource,
	}
}

func (p *lineProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// isLoopbackBaseURL reports whether raw's host is a loopback address
// (127.0.0.1, ::1, or localhost) — the only kind of URL this provider's own
// LINE_API_BASE_URL / LINE_DATA_API_BASE_URL test hooks are meant to accept.
func isLoopbackBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// clientFromProviderData type-asserts providerData (resource.ConfigureRequest.ProviderData)
// into the *client.Client set up in Configure. It's safe to call with a nil
// providerData, which happens when Terraform calls Configure on a resource
// before the provider itself has been configured (e.g. during some
// validate-only flows); in that case it returns nil without an error.
func clientFromProviderData(providerData any) (*client.Client, error) {
	if providerData == nil {
		return nil, nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		return nil, fmt.Errorf("expected *client.Client, got %T", providerData)
	}
	return c, nil
}
