package provider

import (
	"context"
	"os"

	"github.com/aztekas/cleura-client-go/pkg/api/cleura"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &cleuraProvider{}
)

type cleuraProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Token    types.String `tfsdk:"token"`
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &cleuraProvider{
			version: version,
		}
	}
}

// cleuraProvider is the provider implementation.
type cleuraProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *cleuraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cleura"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *cleuraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "Cleura API hostname. Takes CLEURA_API_HOST environment variable if not set.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Cleura cloud username. Takes CLEURA_API_USERNAME environment variable if not set.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "API token used for communication with cleura cloud provider API. Takes CLEURA_API_TOKEN environment variable if not set.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares a Cleura API client for data sources and resources.
func (p *cleuraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Cleura client")
	// Retrieve provider data from configuration
	var config cleuraProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Cleura API Host",
			"The provider cannot create the Cleura API client as there is an unknown configuration value for the Cleura API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CLEURA_API_HOST environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Cleura API Username",
			"The provider cannot create the Cleura API client as there is an unknown configuration value for the Cleura API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CLEURA_API_USERNAME environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Cleura API Username",
			"The provider cannot create the Cleura API client as there is an unknown configuration value for the Cleura API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CLEURA_API_USERNAME environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("CLEURA_API_HOST")
	username := os.Getenv("CLEURA_API_USERNAME")
	token := os.Getenv("CLEURA_API_TOKEN")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Cleura API Host",
			"The provider cannot create the Cleura API client as there is a missing or empty value for the Cleura API host. "+
				"Set the host value in the configuration or use the CLEURA_API_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Cleura API Username",
			"The provider cannot create the Cleura API client as there is a missing or empty value for the Cleura API username. "+
				"Set the username value in the configuration or use the CLEURA_API_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Cleura API TOKEN",
			"The provider cannot create the Cleura API client as there is a missing or empty value for the Cleura API token. "+
				"Set the token value in the configuration or use the CLEURA_API_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "cleura_host", host)
	ctx = tflog.SetField(ctx, "cleura_username", username)
	ctx = tflog.SetField(ctx, "cleura_token", token)

	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "cleura_token")
	tflog.Debug(ctx, "Creating Cleura client")

	// Create a new Cleura client using the configuration values
	client, err := cleura.NewClientNoPassword(&host, &username, &token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Cleura API Client",
			"An unexpected error occurred when creating the Cleura API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Cleura Client Error: "+err.Error(),
		)
		return
	}

	// Make the Cleura client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "Configured Cleura client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *cleuraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewShootClusterDataSource,
		NewShootClusterProfilesDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *cleuraProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewShootClusterResource,
	}
}
