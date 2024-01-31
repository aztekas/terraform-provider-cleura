package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zaikinlv/terraform-provider-cleura/internal/cleura-client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &shootClusterResource{}
	_ resource.ResourceWithConfigure      = &shootClusterResource{}
	_ resource.ResourceWithValidateConfig = &shootClusterResource{}
)

// NewShootClusterResource is a helper function to simplify the provider implementation.
func NewShootClusterResource() resource.Resource {
	return &shootClusterResource{}
}

// shootClusterResource is the resource implementation.
type shootClusterResource struct {
	client *cleura.Client
}

// Configure adds the provider configured client to the resource.
func (r *shootClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *shootClusterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config shootClusterResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	//No validation if no hibernation schedules defined
	if config.HibernationSchedules == nil {
		return
	}

	// Error is either is null
	for i, schedule := range config.HibernationSchedules {
		if schedule.Start.IsNull() || schedule.End.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("hibernation_schedules").AtListIndex(i),
				"Missing Attribute Configuration",
				"Expected both Start and End to be configured ",
			)
		}

	}
	// If nothing matched, return without warning.

}

// Metadata returns the resource type name.
func (r *shootClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shoot_cluster"
}

// Schema defines the schema for the resource.
func (r *shootClusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"kubernetes_version": schema.StringAttribute{
				Required: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"uid": schema.StringAttribute{
				Computed: true,
			},
			"hibernated": schema.BoolAttribute{
				Computed: true,
			},
			"provider_details": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"floating_pool_name": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("ext-net"),
					},
					// "workers_cidr": schema.StringAttribute{
					// 	Optional: true,
					// },
					"worker_groups": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"worker_group_name": schema.StringAttribute{
									Optional: true,
									Computed: true,
								},
								"min_nodes": schema.Int64Attribute{
									Optional: true,
								},
								"max_nodes": schema.Int64Attribute{
									Optional: true,
								},
								"machine_type": schema.StringAttribute{
									Required: true,
								},
								"image_name": schema.StringAttribute{
									Computed: true,
									Optional: true,
									Default:  stringdefault.StaticString("gardenlinux"),
								},
								"image_version": schema.StringAttribute{
									Computed: true,
									Optional: true,
									Default:  stringdefault.StaticString("1312.2.0"),
								},
								"worker_node_volume_size": schema.StringAttribute{
									Computed: true,
									Optional: true,
									Default:  stringdefault.StaticString("50Gi"),
								},
							},
						},
					},
				},
			}, // provider_details end here
			"hibernation_schedules": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Optional: true,
						},
						"end": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

type shootClusterResourceModel struct {
	Timeouts             timeouts.Value             `tfsdk:"timeouts"`
	UID                  types.String               `tfsdk:"uid"`
	Name                 types.String               `tfsdk:"name"`
	Region               types.String               `tfsdk:"region"`
	Project              types.String               `tfsdk:"project"`
	K8sVersion           types.String               `tfsdk:"kubernetes_version"`
	LastUpdated          types.String               `tfsdk:"last_updated"`
	ProviderDetails      shootProviderDetailsModel  `tfsdk:"provider_details"`
	Hibernated           types.Bool                 `tfsdk:"hibernated"`
	HibernationSchedules []hibernationScheduleModel `tfsdk:"hibernation_schedules"`
	//Conditions          []shootClusterConditionsModel          `tfsdk:"conditions"`
	//AdvertisedAddresses []shootClusterAdvertisedAddressesModel `tfsdk:"advertised_addresses"`
}

type hibernationScheduleModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type shootProviderDetailsModel struct {
	FloatingPoolName types.String  `tfsdk:"floating_pool_name"`
	WorkerGroups     []WorkerGroup `tfsdk:"worker_groups"`
}

type WorkerGroup struct {
	WorkerGroupName types.String `tfsdk:"worker_group_name"`
	MachineType     string       `tfsdk:"machine_type"`
	ImageName       types.String `tfsdk:"image_name"`
	ImageVersion    types.String `tfsdk:"image_version"`
	VolumeSize      types.String `tfsdk:"worker_node_volume_size"`
	MinNodes        int16        `tfsdk:"min_nodes"`
	MaxNodes        int16        `tfsdk:"max_nodes"`
}

// Create creates the resource and sets the initial Terraform state.
func (r *shootClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shootClusterResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := plan.Timeouts.Create(ctx, 45*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	var clusterWorkers []cleura.Worker

	for _, worker := range plan.ProviderDetails.WorkerGroups {
		clusterWorkers = append(clusterWorkers, cleura.Worker{

			Name: worker.WorkerGroupName.ValueString(),
			Machine: cleura.MachineDetails{
				Type: worker.MachineType,
				Image: cleura.ImageDetails{
					Name:    worker.ImageName.ValueString(),
					Version: worker.ImageVersion.ValueString(),
				},
			},
			Volume: cleura.VolumeDetails{
				Size: worker.VolumeSize.ValueString(),
			},
		},
		)
	}
	clusterRequest := cleura.ShootClusterRequest{
		Shoot: cleura.ShootClusterRequestConfig{
			Name: plan.Name.ValueString(),
			KubernetesVersion: cleura.K8sVersion{
				Version: plan.K8sVersion.ValueString(),
			},
			Provider: cleura.ProviderDetails{
				InfrastructureConfig: cleura.InfrastructureConfigDetails{
					FloatingPoolName: plan.ProviderDetails.FloatingPoolName.ValueString(),
				},
				Workers: clusterWorkers,
			},
		},
	}
	shootResponse, err := r.client.CreateShootCluster(plan.Region.ValueString(), plan.Project.ValueString(), clusterRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating shoot cluster",
			"Could not create cluster, unexpected error: "+err.Error(),
		)
		return
	}
	//Populating Computed fields
	plan.UID = types.StringValue(shootResponse.Metadata.UID)
	plan.Hibernated = types.BoolValue(shootResponse.Status.Hibernated)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.ProviderDetails.WorkerGroups = []WorkerGroup{}
	for _, worker := range shootResponse.Spec.Provider.Workers {
		plan.ProviderDetails.WorkerGroups = append(plan.ProviderDetails.WorkerGroups, WorkerGroup{
			WorkerGroupName: types.StringValue(worker.Name),
			MachineType:     worker.Machine.Type,
			ImageName:       types.StringValue(worker.Machine.Image.Name),
			ImageVersion:    types.StringValue(worker.Machine.Image.Version),
			VolumeSize:      types.StringValue(worker.Volume.Size),
			MinNodes:        worker.Minimum,
			MaxNodes:        worker.Maximum,
		})
	}

	err = createClusterOperationWaiter(r.client, ctx, createTimeout, plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(

			"API Error Shoot Cluster Resource status check",
			fmt.Sprintf("... details ... %s", err),
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func createClusterOperationWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.MaxInterval = 2 * time.Minute
	b.InitialInterval = 10 * time.Second
	b.Multiplier = 2
	operation := func() error {
		clusterResp, err := client.GetShootCluster(clusterName, clusterRegion, clusterProject)
		if err != nil {
			return backoff.Permanent(err)
		}
		for _, cond := range clusterResp.Status.Conditions {
			if cond.Status != "True" {
				return errors.New("cluster is not ready yet")
			}

		}

		return nil
	}
	return backoff.Retry(operation, backoff.WithContext(b, ctx))

}

func deleteClusterOperationWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.MaxInterval = 2 * time.Minute
	b.InitialInterval = 10 * time.Second
	b.Multiplier = 2
	operation := func() error {

		_, err := client.GetShootCluster(clusterName, clusterRegion, clusterProject)
		if err != nil {
			re, ok := err.(*cleura.RequestAPIError)
			if ok {
				if re.StatusCode == 404 {
					return nil
				}
			}
			return backoff.Permanent(err)
		}

		return errors.New("cluster is not deleted yet")
	}
	return backoff.Retry(operation, backoff.WithContext(b, ctx))

}

// Read refreshes the Terraform state with the latest data.
func (r *shootClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get current state
	var state shootClusterResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Get refreshed shoot cluster from cleura
	shootResponse, err := r.client.GetShootCluster(state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Shoot cluster",
			"Could not read Shoot cluster name "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}
	state.Name = types.StringValue(shootResponse.Metadata.Name)
	state.UID = types.StringValue(shootResponse.Metadata.UID)
	state.Hibernated = types.BoolValue(shootResponse.Status.Hibernated)

	state.ProviderDetails.WorkerGroups = []WorkerGroup{}
	for _, worker := range shootResponse.Spec.Provider.Workers {
		state.ProviderDetails.WorkerGroups = append(state.ProviderDetails.WorkerGroups, WorkerGroup{
			WorkerGroupName: types.StringValue(worker.Name),
			MachineType:     worker.Machine.Type,
			ImageName:       types.StringValue(worker.Machine.Image.Name),
			ImageVersion:    types.StringValue(worker.Machine.Image.Version),
			VolumeSize:      types.StringValue(worker.Volume.Size),
			MinNodes:        worker.Minimum,
			MaxNodes:        worker.Maximum,
		})

	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *shootClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *shootClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state shootClusterResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Set default delete timeout if not set in configuration
	createTimeout, diags := state.Timeouts.Delete(ctx, 45*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing cluster
	_, err := r.client.DeleteShootCluster(state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Shoot Cluster",
			"Could not delete Shoot Cluster, unexpected error: "+err.Error(),
		)
		return
	}
	// Wait until API responds with 404
	err = deleteClusterOperationWaiter(r.client, ctx, createTimeout, state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(

			"API Error Shoot Cluster Resource status check",
			fmt.Sprintf("... details ... %s", err),
		)
		return
	}

}
