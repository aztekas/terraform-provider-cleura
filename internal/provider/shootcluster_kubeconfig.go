package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/aztekas/cleura-client-go/pkg/api/cleura"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &shootClusterKubeconfigResource{}
	_ resource.ResourceWithConfigure      = &shootClusterKubeconfigResource{}
	_ resource.ResourceWithValidateConfig = &shootClusterKubeconfigResource{}
	_ resource.ResourceWithModifyPlan     = &shootClusterKubeconfigResource{}
)

// NewshootClusterKubeconfigResource is a helper function to simplify the provider implementation.
func NewShootClusterKubeconfigResource() resource.Resource {
	return &shootClusterKubeconfigResource{}
}

// shootClusterKubeconfigResource is the resource implementation.
type shootClusterKubeconfigResource struct {
	client *cleura.Client
}

// Configure adds the provider configured client to the resource.
func (r *shootClusterKubeconfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client
}

func (r *shootClusterKubeconfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config shootClusterKubeconfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If nothing matched, return without warning.
}

// Metadata returns the resource type name.
func (r *shootClusterKubeconfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shoot_kubeconfig"
}

// Schema defines the schema for the resource.
func (r *shootClusterKubeconfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the shoot cluster",
			},
			"gardener_domain": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Gardener domain. Defaults to 'public'",
				Default:     stringdefault.StaticString("public"),
			},
			"project": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Id of the project where cluster will be created.",
			},
			"region": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "One of available regions for the cluster. Depends on the enabled domains in the project",
			},
			"duration": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Description: "Set the duration (in seconds) for how long the kubeconfig should be valid",
			},
			"config": schema.StringAttribute{
				Computed:    true,
				Description: "The kubeconfig generated from the API.",
			},
			"generated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The timestamp this resource generated the current kubeconfig",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Version: 0,
	}
}

func (r *shootClusterKubeconfigResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan shootClusterKubeconfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.GeneratedAt.ValueString() != "" {
		generatedAt, err := time.Parse(time.RFC3339, plan.GeneratedAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("failed to parse generated_at", err.Error())
			return
		}

		valid_to := plan.Duration.ValueInt64() + generatedAt.Unix()
		now := time.Now().Unix()
		if now > valid_to {
			plan.GeneratedAt = types.StringUnknown()
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("generated_at"))
			resp.Diagnostics.AddWarning("Kubeconfig expired", "The kubeconfig duration specified has elapsed, resource will be recreated")
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

type shootClusterKubeconfigResourceModel struct {
	Name           types.String `tfsdk:"name"`
	Region         types.String `tfsdk:"region"`
	Project        types.String `tfsdk:"project"`
	GardenerDomain types.String `tfsdk:"gardener_domain"`
	Duration       types.Int64  `tfsdk:"duration"`
	Config         types.String `tfsdk:"config"`
	GeneratedAt    types.String `tfsdk:"generated_at"`
}

// Create creates the resource and sets the initial Terraform state.
func (r *shootClusterKubeconfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shootClusterKubeconfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kubeconfig, err := r.client.GenerateKubeConfig(plan.GardenerDomain.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), plan.Name.ValueString(), plan.Duration.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error generating kubeconfig",
			"Could not generate kubeconfig for Shoot cluster "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.Config = types.StringValue(string(kubeconfig))
	plan.GeneratedAt = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *shootClusterKubeconfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state shootClusterKubeconfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *shootClusterKubeconfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan shootClusterKubeconfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *shootClusterKubeconfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state shootClusterKubeconfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
