package provider

import (
	"context"
	"fmt"

	"github.com/aztekas/cleura-client-go/pkg/api/cleura"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &shootClusterDataSource{}
	_ datasource.DataSourceWithConfigure = &shootClusterDataSource{}
)

// Datasource Model

// coffeesModel maps coffees schema data.
type shootClusterDataSourceModel struct {
	UID                 types.String                           `tfsdk:"uid"`
	Name                types.String                           `tfsdk:"name"`
	Region              types.String                           `tfsdk:"region"`
	Project             types.String                           `tfsdk:"project"`
	Hibernated          types.Bool                             `tfsdk:"hibernated"`
	Conditions          []shootClusterConditionsModel          `tfsdk:"conditions"`
	AdvertisedAddresses []shootClusterAdvertisedAddressesModel `tfsdk:"advertised_addresses"`
}

// coffeesIngredientsModel maps coffee ingredients data
type shootClusterConditionsModel struct {
	Type    types.String `tfsdk:"type"`
	Status  types.String `tfsdk:"status"`
	Message types.String `tfsdk:"message"`
}

type shootClusterAdvertisedAddressesModel struct {
	Name types.String `tfsdk:"name"`
	Url  types.String `tfsdk:"url"`
}

// NewCoffeesDataSource is a helper function to simplify the provider implementation.
func NewShootClusterDataSource() datasource.DataSource {
	return &shootClusterDataSource{}
}

// shootClusterDataSource is the data source implementation.
type shootClusterDataSource struct {
	client *cleura.Client
}

// Metadata returns the data source type name.
func (d *shootClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shoot_cluster"
}

// Schema defines the schema for the data source.
func (d *shootClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"uid": schema.StringAttribute{
				Computed: true,
			},
			"project": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"hibernated": schema.BoolAttribute{
				Computed: true,
			},
			"advertised_addresses": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"url": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"conditions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
						"message": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *shootClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state shootClusterDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := d.client.GetShootCluster(state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Shoot Cluster",
			err.Error(),
		)
		return
	}
	//TODO: Make a function for this mapping
	state.Hibernated = types.BoolValue(cluster.Status.Hibernated)
	state.UID = types.StringValue(cluster.Metadata.UID)
	for _, condition := range cluster.Status.Conditions {
		state.Conditions = append(state.Conditions, shootClusterConditionsModel{
			Type:    types.StringValue(condition.Type),
			Status:  types.StringValue(condition.Status),
			Message: types.StringValue(condition.Message),
		})
	}
	for _, address := range cluster.Status.AdvertisedAddresses {
		state.AdvertisedAddresses = append(state.AdvertisedAddresses, shootClusterAdvertisedAddressesModel{
			Name: types.StringValue(address.Name),
			Url:  types.StringValue(address.Url),
		})
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *shootClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*cleura.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *cleura.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}
