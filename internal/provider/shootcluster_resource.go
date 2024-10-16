package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aztekas/cleura-client-go/pkg/api/cleura"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &shootClusterResource{}
	_ resource.ResourceWithConfigure      = &shootClusterResource{}
	_ resource.ResourceWithValidateConfig = &shootClusterResource{}
	_ resource.ResourceWithImportState    = &shootClusterResource{}
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
	// No validation if no hibernation schedules defined
	if config.HibernationSchedules == nil {
		return
	}

	// Error if either start or end is null
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
				Update: true,
			}),
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the shoot cluster",
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
			"kubernetes_version": schema.StringAttribute{
				Required:    true,
				Description: "One of the currently available Kubernetes versions",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Set local time when cluster resource is created and each time cluster is updated.",
			},
			"uid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Unique cluster ID",
			},
			"hibernated": schema.BoolAttribute{
				Computed:    true,
				Description: "Show current hibernation state of the cluster",
			},
			"provider_details": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Cluster details.",
				Attributes: map[string]schema.Attribute{
					"floating_pool_name": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("ext-net"),
						Description: "The name of the external network to connect to. Defaults to 'ext-net'.",
					},
					"worker_groups": schema.ListNestedAttribute{
						Required:    true,
						Description: "Defines the worker groups",
						Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"worker_group_name": schema.StringAttribute{
									Required:    true,
									Description: "Worker group name. Max 6 lowercase alphanumeric characters.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.RequiresReplaceIf(
											func(ctx context.Context, sr planmodifier.StringRequest, rrifr *stringplanmodifier.RequiresReplaceIfFuncResponse) {
												rrifr.RequiresReplace = !sr.StateValue.IsNull()
											},
											"Requires replace only if modifying existing value", ""),
									},
								},
								"min_nodes": schema.Int64Attribute{
									Required:    true,
									Description: "The minimum number of worker nodes in the worker group.",
								},
								"max_nodes": schema.Int64Attribute{
									Required:    true,
									Description: "The maximum number of worker nodes in the worker group",
								},
								"machine_type": schema.StringAttribute{
									Required:    true,
									Description: "The name of the desired type/flavor of the worker nodes",
								},
								"image_name": schema.StringAttribute{
									Computed:    true,
									Optional:    true,
									Description: "The name of the image of the worker nodes",
									Default:     stringdefault.StaticString("gardenlinux"),
								},
								"image_version": schema.StringAttribute{
									Computed:    true,
									Optional:    true,
									Description: "The version of the image of the worker nodes",
									Default:     stringdefault.StaticString("1312.2.0"),
								},
								"worker_node_volume_size": schema.StringAttribute{
									Computed:    true,
									Optional:    true,
									Description: "The desired size of the volume used for the worker nodes. Example '50Gi'",
									Default:     stringdefault.StaticString("50Gi"),
								},
							},
						},
					},
				},
			}, // provider_details end here
			"hibernation_schedules": schema.ListNestedAttribute{
				Optional:    true,
				Description: "An array containing desired hibernation schedules",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Optional:    true,
							Description: "The time when the hibernation should start in Cron time format",
						},
						"end": schema.StringAttribute{
							Optional:    true,
							Description: "The time when the hibernation should end in Cron time format",
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
	// Conditions          []shootClusterConditionsModel          `tfsdk:"conditions"`
	// AdvertisedAddresses []shootClusterAdvertisedAddressesModel `tfsdk:"advertised_addresses"`
}

type hibernationScheduleModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type shootProviderDetailsModel struct {
	FloatingPoolName types.String       `tfsdk:"floating_pool_name"`
	WorkerGroups     []workerGroupModel `tfsdk:"worker_groups"`
}

type workerGroupModel struct {
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
	tflog.Debug(ctx, "XXX_CREATE")
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

	// Mapping defined workers
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
			Minimum: worker.MinNodes,
			Maximum: worker.MaxNodes,
		},
		)
	}
	// Mapping hibernation schedules
	var hibernationSchedules []cleura.HibernationSchedule
	for _, schedule := range plan.HibernationSchedules {
		hibernationSchedules = append(hibernationSchedules, cleura.HibernationSchedule{
			Start: schedule.Start.ValueString(),
			End:   schedule.End.ValueString(),
		},
		)
	}
	tflog.Debug(ctx, fmt.Sprintf("Here's hibernation schedules: %+v", hibernationSchedules))

	//------------------------------
	clusterRequest := cleura.ShootClusterRequest{
		Shoot: cleura.ShootClusterRequestConfig{
			Name: plan.Name.ValueString(),
			KubernetesVersion: &cleura.K8sVersion{
				Version: plan.K8sVersion.ValueString(),
			},
			Provider: &cleura.ProviderDetails{
				InfrastructureConfig: cleura.InfrastructureConfigDetails{
					FloatingPoolName: plan.ProviderDetails.FloatingPoolName.ValueString(),
				},
				Workers: clusterWorkers,
			},
		},
	}
	tflog.Debug(ctx, fmt.Sprintf("Here's clusterRequest: %+v", clusterRequest))
	// Hibernation must be set to nil(or omitted in clusterRequest) if no schedules defined in config
	if len(hibernationSchedules) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("Hibernation schedules count is: %v", len(hibernationSchedules)))
		clusterRequest.Shoot.Hibernation = &cleura.HibernationSchedules{
			HibernationSchedules: hibernationSchedules,
		}
		tflog.Debug(ctx, "Hibernation schedules are set")
	}

	// Debug clusterRequest content
	jsonByte, err := json.Marshal(clusterRequest)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("error from marshaling clusterRequest: %v", err))
	}
	tflog.Debug(ctx, fmt.Sprintf("clusterRequest: %v", string(jsonByte)))

	shootResponse, err := r.client.CreateShootCluster(plan.Region.ValueString(), plan.Project.ValueString(), clusterRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating shoot cluster",
			"Could not create cluster, unexpected error: "+err.Error(),
		)
		return
	}
	// Populating Computed fields
	plan.UID = types.StringValue(shootResponse.Shoot.UID)
	plan.Hibernated = types.BoolValue(shootResponse.Shoot.Hibernation.Enabled)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.ProviderDetails.WorkerGroups = []workerGroupModel{}
	for _, worker := range shootResponse.Shoot.Provider.Workers {
		plan.ProviderDetails.WorkerGroups = append(plan.ProviderDetails.WorkerGroups, workerGroupModel{
			WorkerGroupName: types.StringValue(worker.Name),
			MachineType:     worker.Machine.Type,
			ImageName:       types.StringValue(worker.Machine.Image.Name),
			ImageVersion:    types.StringValue(worker.Machine.Image.Version),
			VolumeSize:      types.StringValue(worker.Volume.Size),
			MinNodes:        worker.Minimum,
			MaxNodes:        worker.Maximum,
		})
	}

	err = clusterReadyOperationWaiter(r.client, ctx, createTimeout, plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
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

func clusterReconcileWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.InitialInterval = 120 * time.Second
	b.MaxInterval = 75 * time.Second
	b.Multiplier = 2
	operation := func() error {
		clusterResp, err := client.GetShootCluster(clusterName, clusterRegion, clusterProject)
		if err != nil {
			return backoff.Permanent(err)
		}
		lastState := clusterResp.Status.LastOperation.State
		lastOperationType := clusterResp.Status.LastOperation.Type
		if !((lastState == "Succeeded" && lastOperationType == "Create") || (lastState == "Succeeded" && lastOperationType == "Reconcile")) {

			if (b.GetElapsedTime()+b.NextBackOff() >= 400*time.Second) && b.MaxInterval != 30*time.Second {
				b.MaxInterval = 30 * time.Second
				b.InitialInterval = 30 * time.Second
				b.RandomizationFactor = 0
				b.MaxElapsedTime -= b.GetElapsedTime()
				b.Reset()
			}
			return errors.New("last operation is not finished yet")
		}
		return nil
	}
	return backoff.Retry(operation, backoff.WithContext(b, ctx))
}

func clusterReadyOperationWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.MaxInterval = 75 * time.Second
	b.InitialInterval = 120 * time.Second
	b.Multiplier = 2
	operation := func() error {
		clusterResp, err := client.GetShootCluster(clusterName, clusterRegion, clusterProject)
		if err != nil {
			return backoff.Permanent(err)
		}
		for _, cond := range clusterResp.Status.Conditions {
			if cond.Status != "True" {
				if (b.GetElapsedTime()+b.NextBackOff() >= 400*time.Second) && b.MaxInterval != 30*time.Second {
					b.MaxInterval = 30 * time.Second
					b.InitialInterval = 30 * time.Second
					b.RandomizationFactor = 0
					b.MaxElapsedTime -= b.GetElapsedTime()
					b.Reset()
				}
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
	b.MaxInterval = 75 * time.Second
	b.InitialInterval = 120 * time.Second
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
		if (b.GetElapsedTime()+b.NextBackOff() >= 500*time.Second) && b.MaxInterval != 30*time.Second {
			b.MaxInterval = 30 * time.Second
			b.InitialInterval = 30 * time.Second
			b.RandomizationFactor = 0
			b.MaxElapsedTime -= b.GetElapsedTime()
			b.Reset()
		}

		return errors.New("cluster is not deleted yet")
	}
	return backoff.Retry(operation, backoff.WithContext(b, ctx))

}

// Read refreshes the Terraform state with the latest data.
func (r *shootClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "XXX_READ")

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
		re, ok := err.(*cleura.RequestAPIError)
		if ok {
			// Remove resource from state if it was deleted outside terraform
			if re.StatusCode == 404 {
				resp.State.RemoveResource(ctx)
				resp.Diagnostics.AddWarning("Resource has been deleted outside terraform", "New resource will be created")
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Shoot cluster",
			"Could not read Shoot cluster name "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("shootResponse: %+v", shootResponse))

	state.Name = types.StringValue(shootResponse.Metadata.Name)
	state.UID = types.StringValue(shootResponse.Metadata.UID)
	state.Hibernated = types.BoolValue(shootResponse.Status.Hibernated)
	state.K8sVersion = types.StringValue(shootResponse.Spec.Kubernetes.Version)

	state.ProviderDetails.WorkerGroups = []workerGroupModel{}
	for _, worker := range shootResponse.Spec.Provider.Workers {
		state.ProviderDetails.WorkerGroups = append(state.ProviderDetails.WorkerGroups, workerGroupModel{
			WorkerGroupName: types.StringValue(worker.Name),
			MachineType:     worker.Machine.Type,
			ImageName:       types.StringValue(worker.Machine.Image.Name),
			ImageVersion:    types.StringValue(worker.Machine.Image.Version),
			VolumeSize:      types.StringValue(worker.Volume.Size),
			MinNodes:        worker.Minimum,
			MaxNodes:        worker.Maximum,
		})

	}
	tflog.Debug(ctx, fmt.Sprintf("Hibschedules current state: %v", state.HibernationSchedules))
	var hibSchedules []hibernationScheduleModel

	for _, schedule := range shootResponse.Spec.Hibernation.HibernationResponseSchedules {
		hibSchedules = append(hibSchedules, hibernationScheduleModel{
			Start: types.StringValue(schedule.Start),
			End:   types.StringValue(schedule.End),
		})
	}
	state.HibernationSchedules = hibSchedules
	tflog.Debug(ctx, fmt.Sprintf("Hibschedules after state: %v", state.HibernationSchedules))

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *shootClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "XXX_UPDATE")
	var plan shootClusterResourceModel
	var currentState shootClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := plan.Timeouts.Update(ctx, 45*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if !reflect.DeepEqual(plan.HibernationSchedules, currentState.HibernationSchedules) || !reflect.DeepEqual(plan.K8sVersion, currentState.K8sVersion) {
		tflog.Debug(ctx, "Hibernation schedules or K8s version changed")

		hibernationSchedules := []cleura.HibernationSchedule{}
		for _, schedule := range plan.HibernationSchedules {
			hibernationSchedules = append(hibernationSchedules, cleura.HibernationSchedule{
				Start: schedule.Start.ValueString(),
				End:   schedule.End.ValueString(),
			},
			)
		}
		clusterUpdateRequest := cleura.ShootClusterRequest{
			Shoot: cleura.ShootClusterRequestConfig{
				KubernetesVersion: &cleura.K8sVersion{
					Version: plan.K8sVersion.ValueString(),
				},
				Hibernation: &cleura.HibernationSchedules{
					HibernationSchedules: hibernationSchedules,
				},
			},
		}

		_, err := r.client.UpdateShootCluster(plan.Region.ValueString(), plan.Project.ValueString(), plan.Name.ValueString(), clusterUpdateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating shoot cluster",
				"Could not update cluster, unexpected error: "+err.Error(),
			)
			return
		}

	}

	tflog.Debug(ctx, "Workergroups changed")
	wgModify, wgCreate, wgDelete := getCreateModifyDeleteWorkgroups(plan.ProviderDetails.WorkerGroups, currentState.ProviderDetails.WorkerGroups)
	tflog.Debug(ctx, fmt.Sprintf("modify: %+v, create: %+v, delete: %+v, plan: %+v, state: %+v", wgModify, wgCreate, wgDelete, plan.ProviderDetails.WorkerGroups, currentState.ProviderDetails.WorkerGroups))
	for _, wg := range wgModify {
		worker := cleura.WorkerGroupRequest{
			Worker: cleura.Worker{
				Minimum: wg.MinNodes,
				Maximum: wg.MaxNodes,
				Machine: cleura.MachineDetails{
					Type: wg.MachineType,
					Image: cleura.ImageDetails{
						Name:    wg.ImageName.ValueString(),
						Version: wg.ImageVersion.ValueString(),
					},
				},
				Volume: cleura.VolumeDetails{
					Size: wg.VolumeSize.ValueString(),
				},
			},
		}
		_, err := r.client.UpdateWorkerGroup(plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), wg.WorkerGroupName.ValueString(), worker)
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error Updating Worker Group",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

	}
	for _, wg := range wgCreate {
		worker := cleura.WorkerGroupRequest{
			Worker: cleura.Worker{
				Name:    wg.WorkerGroupName.ValueString(),
				Minimum: wg.MinNodes,
				Maximum: wg.MaxNodes,
				Machine: cleura.MachineDetails{
					Type: wg.MachineType,
					Image: cleura.ImageDetails{
						Name:    wg.ImageName.ValueString(),
						Version: wg.ImageVersion.ValueString(),
					},
				},
				Volume: cleura.VolumeDetails{
					Size: wg.VolumeSize.ValueString(),
				},
			},
		}
		_, err := r.client.AddWorkerGroup(plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), worker)
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error Adding Worker Group",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

	}
	for _, wg := range wgDelete {

		_, err := r.client.DeleteWorkerGroup(plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), wg.WorkerGroupName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error Deleting Worker Group",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

	}
	// Wait cluster ready after update
	err := clusterReconcileWaiter(r.client, ctx, createTimeout, plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(

			"API Error Shoot Cluster Resource status check",
			fmt.Sprintf("... details ... %s", err),
		)
		return
	}
	clusterUpdateResp, err := r.client.GetShootCluster(plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(

			"API Error Shoot Cluster Resource status check",
			fmt.Sprintf("... details ... %s", err),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("response after all: %+v, ", clusterUpdateResp))

	plan.UID = currentState.UID // types.StringValue(clusterUpdateResp.Metadata.UID)
	plan.Hibernated = types.BoolValue(clusterUpdateResp.Status.Hibernated)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Required for populating computed values
	plan.ProviderDetails.WorkerGroups = []workerGroupModel{}
	for _, worker := range clusterUpdateResp.Spec.Provider.Workers {
		plan.ProviderDetails.WorkerGroups = append(plan.ProviderDetails.WorkerGroups, workerGroupModel{
			WorkerGroupName: types.StringValue(worker.Name),
			MachineType:     worker.Machine.Type,
			ImageName:       types.StringValue(worker.Machine.Image.Name),
			ImageVersion:    types.StringValue(worker.Machine.Image.Version),
			VolumeSize:      types.StringValue(worker.Volume.Size),
			MinNodes:        worker.Minimum,
			MaxNodes:        worker.Maximum,
		})
	}

	var hibSchedules []hibernationScheduleModel // nil

	for _, schedule := range clusterUpdateResp.Spec.Hibernation.HibernationResponseSchedules {
		hibSchedules = append(hibSchedules, hibernationScheduleModel{
			Start: types.StringValue(schedule.Start),
			End:   types.StringValue(schedule.End),
		})
	}
	plan.HibernationSchedules = hibSchedules // set nil if no schedule present, slice with schedules if present

	// Setting the final state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *shootClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "XXX_DELETE")
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

func getCreateModifyDeleteWorkgroups(wgsPlan []workerGroupModel, wgsState []workerGroupModel) (wgModify []workerGroupModel, wgCreate []workerGroupModel, wgDelete []workerGroupModel) {
	stateMap := make(map[string]workerGroupModel)
	for i, wg := range wgsState {
		stateMap[wg.WorkerGroupName.ValueString()] = wgsState[i]
	}
	planMap := make(map[string]workerGroupModel)
	for i, wg := range wgsPlan {
		planMap[wg.WorkerGroupName.ValueString()] = wgsPlan[i]
	}
	for k, v := range planMap {
		if _, ok := stateMap[k]; ok {
			// wg already exists in state, so check it is modified
			if !reflect.DeepEqual(planMap[k], stateMap[k]) {
				// wgs are different so use the one from the plan
				wgModify = append(wgModify, v)
			}
		} else {
			// wg doesn't exist, so add a new workgroup
			wgCreate = append(wgCreate, v)
		}
	}
	for k, v := range stateMap {
		if _, ok := planMap[k]; !ok {
			// wg doesn't exist in plan, so delete it
			wgDelete = append(wgDelete, v)
		}
	}
	return wgModify, wgCreate, wgDelete

}

func (r *shootClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")
	var state shootClusterResourceModel
	tflog.Debug(ctx, fmt.Sprintf("idparts: %v", idParts))
	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: Name,Region,Project_id. Got: %q", req.ID),
		)
		return
	}
	state.Name = types.StringValue(idParts[0])
	state.Region = types.StringValue(idParts[1])
	state.Project = types.StringValue(idParts[2])

	// Get refreshed shoot cluster from cleura
	shootResponse, err := r.client.GetShootCluster(idParts[0], idParts[1], idParts[2])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Shoot cluster",
			"Could not read Shoot cluster name "+idParts[0]+": "+err.Error(),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("shootResponse: %+v", shootResponse))

	state.UID = types.StringValue(shootResponse.Metadata.UID)
	state.Hibernated = types.BoolValue(shootResponse.Status.Hibernated)
	state.K8sVersion = types.StringValue(shootResponse.Spec.Kubernetes.Version)
	state.ProviderDetails.FloatingPoolName = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.FloatingPoolName)

	state.ProviderDetails.WorkerGroups = []workerGroupModel{}
	for _, worker := range shootResponse.Spec.Provider.Workers {
		state.ProviderDetails.WorkerGroups = append(state.ProviderDetails.WorkerGroups, workerGroupModel{
			WorkerGroupName: types.StringValue(worker.Name),
			MachineType:     worker.Machine.Type,
			ImageName:       types.StringValue(worker.Machine.Image.Name),
			ImageVersion:    types.StringValue(worker.Machine.Image.Version),
			VolumeSize:      types.StringValue(worker.Volume.Size),
			MinNodes:        worker.Minimum,
			MaxNodes:        worker.Maximum,
		})
	}
	tflog.Debug(ctx, fmt.Sprintf("Hibschedules current state: %v", state.HibernationSchedules))
	var hibSchedules []hibernationScheduleModel

	for _, schedule := range shootResponse.Spec.Hibernation.HibernationResponseSchedules {
		hibSchedules = append(hibSchedules, hibernationScheduleModel{
			Start: types.StringValue(schedule.Start),
			End:   types.StringValue(schedule.End),
		})
	}
	state.HibernationSchedules = hibSchedules
	tflog.Debug(ctx, fmt.Sprintf("Hibschedules after state: %v", state.HibernationSchedules))

	state.Timeouts = timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"delete": types.StringType,
			"update": types.StringType,
		}),
	}

	// Set refreshed state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
