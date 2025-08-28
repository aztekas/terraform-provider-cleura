package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/aztekas/cleura-client-go/pkg/api/cleura"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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
	_ resource.ResourceWithUpgradeState   = &shootClusterResource{}
	_ resource.ResourceWithModifyPlan     = &shootClusterResource{}
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
	var config shootClusterResourceModelV1
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

	var diags diag.Diagnostics
	maintenance := attrValuesToMaintenanceModelV0(config.Maintenance, &resp.Diagnostics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if (!maintenance.TimeWindowBegin.IsNull() && maintenance.TimeWindowEnd.IsNull()) ||
		(maintenance.TimeWindowBegin.IsNull() && !maintenance.TimeWindowEnd.IsNull()) {
		resp.Diagnostics.AddError(
			"Missing Attribute Configuration",
			"Both `time_window_begin` and ´time_window_end´ must be set in the configuration.",
		)
	} else if maintenance.TimeWindowBegin.Equal(maintenance.TimeWindowEnd) && maintenance.TimeWindowBegin.ValueString() != "" {
		resp.Diagnostics.AddError(
			"Invalid Attribute Configuration",
			"`time_window_begin` and ´time_window_end´ can not be equal.",
		)
	}

	if (!config.ProviderDetails.NetworkId.IsNull() && config.ProviderDetails.RouterId.IsNull()) ||
		(config.ProviderDetails.NetworkId.IsNull() && !config.ProviderDetails.RouterId.IsNull()) {
		resp.Diagnostics.AddError(
			"Missing Attribute Configuration",
			"Both `network_id` and `router_id` must be set in the configuration.",
		)
	}

	nameRegex := regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?`)
	// Convert elements to Objects
	for _, group := range config.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		worker := attrValuesToWorkerGroupModelV1(objVal, &resp.Diagnostics)
		if !nameRegex.Match([]byte(worker.WorkerGroupName.ValueString())) || len(worker.WorkerGroupName.ValueString()) > 6 {
			resp.Diagnostics.AddError(
				"Invalid Worker Group Name",
				"Worker group names must: \n\t1. Only contain lowercase aplhanumeric characters and hyphens\n\t2. Not begin with hyphen or a number\n\t3. Not end with a hyphen \n\t4. Not be longer than 6 characters",
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
			"kubernetes_version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
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
					"network_id": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The id of the internal OpenStack network to connect worker nodes to. Requires replace if modified.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"router_id": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The id of the OpenStack router to connect the worker subnet to. Requires replace if modified.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"worker_cidr": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The CIDR to use for worker nodes. Cannot overlap with existing subnets in the selected network. Requires replace if modified.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
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
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"worker_node_volume_size": schema.StringAttribute{
									Computed:    true,
									Optional:    true,
									Description: "The desired size of the volume used for the worker nodes. Example '50Gi'",
									Default:     stringdefault.StaticString("50Gi"),
								},
								"annotations": schema.MapAttribute{
									Optional:    true,
									Computed:    true,
									Description: "Annotations for taints nodes",
									ElementType: types.StringType,
									PlanModifiers: []planmodifier.Map{
										mapplanmodifier.UseStateForUnknown(),
									},
									Default: mapdefault.StaticValue(types.MapNull(types.StringType)),
								},
								"labels": schema.MapAttribute{
									Optional:    true,
									Computed:    true,
									Description: "Labels for worker nodes",
									ElementType: types.StringType,
									PlanModifiers: []planmodifier.Map{
										mapplanmodifier.UseStateForUnknown(),
									},
									Default: mapdefault.StaticValue(types.MapNull(types.StringType)),
								},
								"taints": schema.ListNestedAttribute{
									Optional:    true,
									Description: "Taints for worker nodes",
									CustomType:  types.ListType{ElemType: types.ObjectType{AttrTypes: taintAttrTypesV0()}},
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"key": schema.StringAttribute{
												Required:    true,
												Description: "Key name for taint. Must adhere to Kubernetes key naming specifications",
											},
											"value": schema.StringAttribute{
												Required:    true,
												Description: "Value for taint. Must be within Kubernetes taint value specifications",
											},
											"effect": schema.StringAttribute{
												Required:    true,
												Description: "Effect for taint. Possible values are 'NoExecute', 'NoSchedule' and 'PreferNoSchedule'",
												Validators:  []validator.String{stringvalidator.OneOf("NoSchedule", "NoExecute", "PreferNoSchedule")},
											},
										},
									},
									PlanModifiers: []planmodifier.List{
										listplanmodifier.UseStateForUnknown(),
									},
								},
								"zones": schema.ListAttribute{
									Computed:    true,
									Optional:    true,
									Description: "List of availability zones worker nodes can be scheduled in. Defaults to ['nova']",
									ElementType: types.StringType,
									PlanModifiers: []planmodifier.List{
										listplanmodifier.UseStateForUnknown(),
									},
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
			"maintenance": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Configure maintenance properties",
				Attributes: map[string]schema.Attribute{
					"auto_update_kubernetes": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Toggle wether or not to allow automatic kubernetes upgrades. Defaults to 'true'",
						Default:     booldefault.StaticBool(true),
					},
					"auto_update_machine_image": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Toggle wether or not to allow automatic machine image upgrades. Defaults to 'true'",
						Default:     booldefault.StaticBool(true),
					},
					"time_window_begin": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Set when time windows for upgrades should begin, defaults to '000000+0100'",
						Default:     stringdefault.StaticString("000000+0100"),
					},
					"time_window_end": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Set when time windows for upgrades should end, defaults to '010000+0100'",
						Default:     stringdefault.StaticString("010000+0100"),
					},
				},
			},
		},
		Version: 3,
	}
}

func (r *shootClusterResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 0 (prior state version) to 1 (Schema.Version)
		0: {
			PriorSchema: &schema.Schema{
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
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData shootClusterResourceModelV0

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)

				if resp.Diagnostics.HasError() {
					return
				}

				clusterResp, err := r.client.GetShootCluster(
					"public",
					priorStateData.Name.ValueString(),
					priorStateData.Region.ValueString(),
					priorStateData.Project.ValueString(),
				)
				if err != nil {
					resp.Diagnostics.AddError("Failed to get shoot cluster during state upgrade", err.Error())
					return
				}

				runtimeWorkerGroup := make(map[string]cleura.WorkerUpdateResponse)
				for _, worker := range clusterResp.Spec.Provider.Workers {
					runtimeWorkerGroup[worker.Name] = worker
				}

				var workerGroups []attr.Value
				for _, group := range priorStateData.ProviderDetails.WorkerGroups.Elements() {
					objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV0(), group)
					resp.Diagnostics.Append(diags...)
					old := attrValuesToWorkerGroupModelV1(objVal, &resp.Diagnostics)

					// If there is an error
					if err != nil {
						resp.Diagnostics.AddError("Could not find shoot cluster", fmt.Sprintf("failed to retrieve metadata for shoot cluster '%s', error: %s", priorStateData.Name.ValueString(), err))
					}

					annotations, diags := types.MapValueFrom(ctx, types.StringType, runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Annotations)
					resp.Diagnostics.Append(diags...)

					labels, diags := types.MapValueFrom(ctx, types.StringType, runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Labels)
					resp.Diagnostics.Append(diags...)

					var taintAttrValues []attr.Value
					for _, taint := range runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Taints {
						tmp, diags := types.ObjectValueFrom(ctx, taintAttrTypesV0(), Taint{
							Key:    types.StringValue(taint.Key),
							Value:  types.StringValue(taint.Value),
							Effect: types.StringValue(taint.Effect),
						})

						resp.Diagnostics.Append(diags...)
						if resp.Diagnostics.HasError() {
							return
						}

						taintAttrValues = append(taintAttrValues, tmp)
					}

					taints, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: taintAttrTypesV0()}, taintAttrValues)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					zones, diags := types.ListValueFrom(ctx, types.StringType, runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Zones)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					newWg, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), workerGroupModelV1{
						WorkerGroupName: old.WorkerGroupName,
						MachineType:     old.MachineType,
						ImageName:       old.ImageName,
						ImageVersion:    old.ImageVersion,
						VolumeSize:      old.VolumeSize,
						MinNodes:        old.MinNodes,
						MaxNodes:        old.MaxNodes,
						Annotations:     annotations,
						Labels:          labels,
						Taints:          taints,
						Zones:           zones,
					})
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					workerGroups = append(workerGroups, newWg)

				}

				// Set upgraded WorkerGroupModelV1
				var diags diag.Diagnostics
				priorStateData.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, shootClusterResourceModelV1{
					Timeouts:             priorStateData.Timeouts,
					UID:                  priorStateData.UID,
					Region:               priorStateData.Region,
					Project:              priorStateData.Project,
					K8sVersion:           priorStateData.K8sVersion,
					LastUpdated:          priorStateData.LastUpdated,
					GardenerDomain:       types.StringValue("public"),
					ProviderDetails:      priorStateData.ProviderDetails,
					Hibernated:           priorStateData.Hibernated,
					HibernationSchedules: priorStateData.HibernationSchedules,
				})...)
			},
		},
		2: {
			PriorSchema: &schema.Schema{
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
					"kubernetes_version": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
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
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
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
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData shootClusterResourceModelV1

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)

				if resp.Diagnostics.HasError() {
					return
				}

				clusterResp, err := r.client.GetShootCluster(
					priorStateData.GardenerDomain.ValueString(),
					priorStateData.Name.ValueString(),
					priorStateData.Region.ValueString(),
					priorStateData.Project.ValueString(),
				)

				runtimeWorkerGroup := make(map[string]cleura.WorkerUpdateResponse)
				for _, worker := range clusterResp.Spec.Provider.Workers {
					runtimeWorkerGroup[worker.Name] = worker
				}

				var workerGroups []attr.Value
				for _, group := range priorStateData.ProviderDetails.WorkerGroups.Elements() {
					objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV0(), group)
					resp.Diagnostics.Append(diags...)
					old := attrValuesToWorkerGroupModelV1(objVal, &resp.Diagnostics)

					// If there is an error
					if err != nil {
						resp.Diagnostics.AddError("Could not find shoot cluster", fmt.Sprintf("failed to retrieve metadata for shoot cluster '%s', error: %s", priorStateData.Name.ValueString(), err))
					}

					annotations, diags := types.MapValueFrom(ctx, types.StringType, runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Annotations)
					resp.Diagnostics.Append(diags...)

					labels, diags := types.MapValueFrom(ctx, types.StringType, runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Labels)
					resp.Diagnostics.Append(diags...)

					var taintAttrValues []attr.Value
					for _, taint := range runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Taints {
						tmp, diags := types.ObjectValueFrom(ctx, taintAttrTypesV0(), Taint{
							Key:    types.StringValue(taint.Key),
							Value:  types.StringValue(taint.Value),
							Effect: types.StringValue(taint.Effect),
						})

						resp.Diagnostics.Append(diags...)
						if resp.Diagnostics.HasError() {
							return
						}

						taintAttrValues = append(taintAttrValues, tmp)
					}

					taints, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: taintAttrTypesV0()}, taintAttrValues)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					zones, diags := types.ListValueFrom(ctx, types.StringType, runtimeWorkerGroup[old.WorkerGroupName.ValueString()].Zones)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					newWg, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), workerGroupModelV1{
						WorkerGroupName: old.WorkerGroupName,
						MachineType:     old.MachineType,
						ImageName:       old.ImageName,
						ImageVersion:    old.ImageVersion,
						VolumeSize:      old.VolumeSize,
						MinNodes:        old.MinNodes,
						MaxNodes:        old.MaxNodes,
						Annotations:     annotations,
						Labels:          labels,
						Taints:          taints,
						Zones:           zones,
					})
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					workerGroups = append(workerGroups, newWg)

				}

				// Set upgraded WorkerGroupModelV1
				var diags diag.Diagnostics
				priorStateData.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, priorStateData)...)
			},
		},
	}
}

func (r *shootClusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {

	// No plan modification is needed if destroying the resource
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan shootClusterResourceModelV1
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the cloud profile from the API
	profile, err := r.client.GetCloudProfile(plan.GardenerDomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get profile data",
			err.Error(),
		)
		return
	}

	// Use the latest Kubernetes version if not set explicitly
	if plan.K8sVersion.ValueString() == "" {
		plan.K8sVersion = getLatestK8sVersion(profile)
	}

	// If not specified by the user, use all availability zones in the given region
	var availabilityZones []string
	for _, region := range profile.Spec.Regions {
		if region.Name == plan.Region.ValueString() {
			for _, zone := range region.Zones {
				availabilityZones = append(availabilityZones, zone.Name)
			}
		}
	}

	// Convert elements to Objects
	var workerGroups []attr.Value
	for _, group := range plan.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		workerGroups = append(workerGroups, objVal)
	}

	// Iterate over all worker groups and set Kubernetes version to latest if not specified
	for i, wg := range workerGroups {
		worker := attrValuesToWorkerGroupModelV1(wg, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		// Use the latest GardenLinux image if not set explicitly
		if worker.ImageVersion.ValueString() == "" {
			worker.ImageVersion = getLatestGardenlinuxVersion(profile)
		}

		// Set all availability zones if not set explicitly
		if worker.Zones.IsUnknown() {
			zones, err := types.ListValueFrom(ctx, types.StringType, availabilityZones)
			resp.Diagnostics.Append(err...)
			if resp.Diagnostics.HasError() {
				return
			}
			worker.Zones = zones
		}

		workerGroups[i], diags = types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), worker)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	maintenance := attrValuesToMaintenanceModelV0(plan.Maintenance, &resp.Diagnostics)

	if k8s := maintenance.AutoUpdateKubernetes; k8s.IsNull() || k8s.IsUnknown() {
		maintenance.AutoUpdateKubernetes = types.BoolValue(true)
	}

	if machineImage := maintenance.AutoUpdateMachineImage; machineImage.IsNull() || machineImage.IsUnknown() {
		maintenance.AutoUpdateMachineImage = types.BoolValue(true)
	}

	if begin := maintenance.TimeWindowBegin; begin.IsNull() || begin.IsUnknown() {
		maintenance.TimeWindowBegin = types.StringValue("000000+0100")
	}

	if end := maintenance.TimeWindowEnd; end.IsNull() || end.IsUnknown() {
		maintenance.TimeWindowEnd = types.StringValue("010000+0100")
	}

	plan.Maintenance, diags = types.ObjectValueFrom(ctx, maintenanceAttrTypesV0(), maintenance)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the updated objects to the plan
	plan.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Plan.Set(ctx, plan)
}

type shootClusterResourceModelV0 struct {
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

type shootClusterResourceModelV1 struct {
	Timeouts             timeouts.Value             `tfsdk:"timeouts"`
	UID                  types.String               `tfsdk:"uid"`
	Name                 types.String               `tfsdk:"name"`
	Region               types.String               `tfsdk:"region"`
	Project              types.String               `tfsdk:"project"`
	K8sVersion           types.String               `tfsdk:"kubernetes_version"`
	LastUpdated          types.String               `tfsdk:"last_updated"`
	GardenerDomain       types.String               `tfsdk:"gardener_domain"`
	ProviderDetails      shootProviderDetailsModel  `tfsdk:"provider_details"`
	Hibernated           types.Bool                 `tfsdk:"hibernated"`
	HibernationSchedules []hibernationScheduleModel `tfsdk:"hibernation_schedules"`
	Maintenance          types.Object               `tfsdk:"maintenance"`
	// Conditions          []shootClusterConditionsModel          `tfsdk:"conditions"`
	// AdvertisedAddresses []shootClusterAdvertisedAddressesModel `tfsdk:"advertised_addresses"`
}

type hibernationScheduleModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type maintenanceModel struct {
	AutoUpdateKubernetes   types.Bool   `tfsdk:"auto_update_kubernetes"`
	AutoUpdateMachineImage types.Bool   `tfsdk:"auto_update_machine_image"`
	TimeWindowBegin        types.String `tfsdk:"time_window_begin"`
	TimeWindowEnd          types.String `tfsdk:"time_window_end"`
}

type shootProviderDetailsModel struct {
	FloatingPoolName types.String `tfsdk:"floating_pool_name"`
	NetworkId        types.String `tfsdk:"network_id"`
	RouterId         types.String `tfsdk:"router_id"`
	WorkerCidr       types.String `tfsdk:"worker_cidr"`
	WorkerGroups     types.List   `tfsdk:"worker_groups"`
}

type workerGroupModelV1 struct {
	WorkerGroupName types.String `tfsdk:"worker_group_name"`
	MachineType     types.String `tfsdk:"machine_type"`
	ImageName       types.String `tfsdk:"image_name"`
	ImageVersion    types.String `tfsdk:"image_version"`
	VolumeSize      types.String `tfsdk:"worker_node_volume_size"`
	MinNodes        types.Int64  `tfsdk:"min_nodes"`
	MaxNodes        types.Int64  `tfsdk:"max_nodes"`
	Annotations     types.Map    `tfsdk:"annotations"`
	Labels          types.Map    `tfsdk:"labels"`
	Taints          types.List   `tfsdk:"taints"`
	Zones           types.List   `tfsdk:"zones"`
}

type KeyValuePair struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type Taint struct {
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
	Effect types.String `tfsdk:"effect"`
}

func workerGroupModelAttrTypesV0() map[string]attr.Type {
	return map[string]attr.Type{
		"worker_group_name":       types.StringType,
		"machine_type":            types.StringType,
		"image_name":              types.StringType,
		"image_version":           types.StringType,
		"worker_node_volume_size": types.StringType,
		"min_nodes":               types.Int64Type,
		"max_nodes":               types.Int64Type,
	}
}

func workerGroupModelAttrTypesV1() map[string]attr.Type {
	return map[string]attr.Type{
		"worker_group_name":       types.StringType,
		"machine_type":            types.StringType,
		"image_name":              types.StringType,
		"image_version":           types.StringType,
		"worker_node_volume_size": types.StringType,
		"min_nodes":               types.Int64Type,
		"max_nodes":               types.Int64Type,
		"annotations":             types.MapType{ElemType: types.StringType},
		"labels":                  types.MapType{ElemType: types.StringType},
		"taints":                  types.ListType{ElemType: types.ObjectType{AttrTypes: taintAttrTypesV0()}},
		"zones":                   types.ListType{ElemType: types.StringType},
	}
}

func taintAttrTypesV0() map[string]attr.Type {
	return map[string]attr.Type{
		"key":    types.StringType,
		"value":  types.StringType,
		"effect": types.StringType,
	}
}

func maintenanceAttrTypesV0() map[string]attr.Type {
	return map[string]attr.Type{
		"auto_update_kubernetes":    types.BoolType,
		"auto_update_machine_image": types.BoolType,
		"time_window_begin":         types.StringType,
		"time_window_end":           types.StringType,
	}
}

func int16Downcast(num int64) (int16, error) {
	if num > math.MaxInt16 || num < math.MinInt16 {
		return 0, fmt.Errorf("value %d cannot be downcasted to int16 as it would overflow", num)
	}

	return int16(num), nil
}

func getStringAttr(key string, value attr.Value) (types.String, diag.Diagnostics) {
	objVal, ok := value.(types.Object)
	var diags diag.Diagnostics

	if !ok {
		diags.AddError("Invalid Value", "Expected Object in list")
		return types.StringNull(), diags
	}

	attributes := objVal.Attributes()
	if attribute, ok := attributes[key].(types.String); ok {
		return attribute, nil
	} else {
		return types.StringNull(), nil
	}
}

func getInt64Attr(key string, value attr.Value) (types.Int64, diag.Diagnostics) {
	objVal, ok := value.(types.Object)
	var diags diag.Diagnostics

	if !ok {
		diags.AddError("Invalid Value", "Expected Object in list")
		return types.Int64Null(), diags
	}

	attributes := objVal.Attributes()
	if attribute, ok := attributes[key].(types.Int64); ok {
		return attribute, nil
	} else {
		return types.Int64Null(), nil
	}
}

func getBoolAttr(key string, value attr.Value) (types.Bool, diag.Diagnostics) {
	objVal, ok := value.(types.Object)
	var diags diag.Diagnostics

	if !ok {
		diags.AddError("Invalid Value", "Expected Object in list")
		return types.BoolNull(), diags
	}

	attributes := objVal.Attributes()
	if attribute, ok := attributes[key].(types.Bool); ok {
		return attribute, nil
	} else {
		return types.BoolNull(), nil
	}
}

func getStringMapAttr(key string, value attr.Value) (types.Map, diag.Diagnostics) {
	objVal, ok := value.(types.Object)
	var diags diag.Diagnostics

	if !ok {
		diags.AddError("Invalid Value", "Expected Object in list")
		return types.MapNull(types.StringType), diags
	}

	attributes := objVal.Attributes()
	if attribute, ok := attributes[key].(types.Map); ok {
		return attribute, nil
	} else {
		return types.MapNull(types.StringType), nil
	}
}

func getStringListAttr(key string, value attr.Value) (types.List, diag.Diagnostics) {
	objVal, ok := value.(types.Object)
	var diags diag.Diagnostics

	if !ok {
		diags.AddError("Invalid Value", "Expected Object in list")
		return types.ListNull(types.StringType), diags
	}

	attributes := objVal.Attributes()
	if attribute, ok := attributes[key].(types.List); ok {
		return attribute, nil
	} else {
		return types.ListNull(types.StringType), nil
	}
}

func getTaintListAttr(key string, value attr.Value) (types.List, diag.Diagnostics) {
	objVal, ok := value.(types.Object)
	var diags diag.Diagnostics

	if !ok {
		diags.AddError("Invalid Value", "Expected Object in list")
		return types.ListNull(types.ObjectType{AttrTypes: taintAttrTypesV0()}), diags
	}

	attributes := objVal.Attributes()
	if attribute, ok := attributes[key].(types.List); ok {
		return attribute, nil
	} else {
		return types.ListNull(types.ObjectType{AttrTypes: taintAttrTypesV0()}), nil
	}
}

func attrValuesToWorkerGroupModelV1(value attr.Value, diags *diag.Diagnostics) workerGroupModelV1 {
	var err diag.Diagnostics
	workerGroup := workerGroupModelV1{}

	workerGroup.WorkerGroupName, err = getStringAttr("worker_group_name", value)
	diags.Append(err...)

	workerGroup.MachineType, err = getStringAttr("machine_type", value)
	diags.Append(err...)

	workerGroup.ImageName, err = getStringAttr("image_name", value)
	diags.Append(err...)

	workerGroup.ImageVersion, err = getStringAttr("image_version", value)
	diags.Append(err...)

	workerGroup.VolumeSize, err = getStringAttr("worker_node_volume_size", value)
	diags.Append(err...)

	workerGroup.MinNodes, err = getInt64Attr("min_nodes", value)
	diags.Append(err...)

	workerGroup.MaxNodes, err = getInt64Attr("max_nodes", value)
	diags.Append(err...)

	workerGroup.Annotations, err = getStringMapAttr("annotations", value)
	diags.Append(err...)

	workerGroup.Labels, err = getStringMapAttr("labels", value)
	diags.Append(err...)

	workerGroup.Taints, err = getTaintListAttr("taints", value)
	diags.Append(err...)

	workerGroup.Zones, err = getStringListAttr("zones", value)
	diags.Append(err...)

	return workerGroup
}

func attrValuesToWorkerGroupModelSlice(values []attr.Value, diags *diag.Diagnostics) []workerGroupModelV1 {
	var result []workerGroupModelV1

	for _, val := range values {
		result = append(result, attrValuesToWorkerGroupModelV1(val, diags))
	}

	return result
}

func attrValuesToMaintenanceModelV0(value attr.Value, diags *diag.Diagnostics) maintenanceModel {
	var err diag.Diagnostics
	maintenance := maintenanceModel{}

	maintenance.AutoUpdateKubernetes, err = getBoolAttr("auto_update_kubernetes", value)
	diags.Append(err...)

	maintenance.AutoUpdateMachineImage, err = getBoolAttr("auto_update_machine_image", value)
	diags.Append(err...)

	maintenance.TimeWindowBegin, err = getStringAttr("time_window_begin", value)
	diags.Append(err...)

	maintenance.TimeWindowEnd, err = getStringAttr("time_window_end", value)
	diags.Append(err...)

	return maintenance
}

func mapValueToStringMap(source types.Map, diags *diag.Diagnostics) map[string]string {

	// Check if the map is null or unknown
	if source.IsNull() || source.IsUnknown() {
		return nil
	}

	// Convert attribute value map to native map[string]string
	result := make(map[string]string, len(source.Elements()))
	for key, val := range source.Elements() {
		strVal, ok := val.(types.String)
		if !ok {
			diags.AddError("Map value is not a string", fmt.Sprintf("map value for key '%s' is not a string: %T", key, val))
			return nil
		}
		if strVal.IsUnknown() || strVal.IsNull() {
			continue
		}
		result[key] = strVal.ValueString()
	}

	return result
}

func listValueToTaintList(ctx context.Context, source types.List, diags *diag.Diagnostics) []Taint {
	var taints []Taint
	for _, taint := range source.Elements() {
		objVal, err := types.ObjectValueFrom(ctx, taintAttrTypesV0(), taint)
		diags.Append(err...)

		key, err := getStringAttr("key", objVal)
		diags.Append(err...)

		value, err := getStringAttr("value", objVal)
		diags.Append(err...)

		effect, err := getStringAttr("effect", objVal)
		diags.Append(err...)

		taints = append(taints, Taint{
			Key:    key,
			Value:  value,
			Effect: effect,
		})
	}

	return taints
}

func listStringToStringSlice(source types.List, diags *diag.Diagnostics) []string {
	var result []string
	for _, item := range source.Elements() {
		strVal, ok := item.(types.String)
		if !ok {
			diags.AddError(
				"Unexpected Type",
				"Element in list was not of type types.String",
			)
			return nil
		}
		if strVal.IsUnknown() || strVal.IsNull() {
			diags.AddError(
				"Unknown or null values in list",
				"Expected all values to be known or not null",
			)
			return nil
		}

		result = append(result, strVal.ValueString())
	}

	return result
}

func cleuraTaintListToTaintList(source []cleura.Taint) []Taint {
	var result []Taint

	for _, t := range source {
		result = append(result, Taint{
			Key:    types.StringValue(t.Key),
			Value:  types.StringValue(t.Value),
			Effect: types.StringValue(t.Effect),
		})
	}

	return result
}

func createWorkerRequestV1(ctx context.Context, workerGroup workerGroupModelV1) (cleura.WorkerRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	var request cleura.WorkerRequest

	minNodes, err := int16Downcast(workerGroup.MinNodes.ValueInt64())
	// Downcast min_nodes from Terraform Int64 type
	if err != nil {
		diags.AddError("Could not downcast min_nodes value", err.Error())
		return cleura.WorkerRequest{}, diags
	}

	// Downcast max_nodes from Terraform Int64 type
	maxNodes, err := int16Downcast(workerGroup.MaxNodes.ValueInt64())
	if err != nil {
		diags.AddError("Could not downcast max_nodes value", err.Error())
		return cleura.WorkerRequest{}, diags
	}

	annotations := make([]cleura.KeyValuePair, 0)
	annotationsMap := mapValueToStringMap(workerGroup.Annotations, &diags)
	if diags.HasError() {
		return cleura.WorkerRequest{}, diags
	}
	for key, value := range annotationsMap {
		annotations = append(annotations, cleura.KeyValuePair{
			Key:   key,
			Value: value,
		})
	}

	labels := make([]cleura.KeyValuePair, 0)
	labelsMap := mapValueToStringMap(workerGroup.Labels, &diags)
	if diags.HasError() {
		return cleura.WorkerRequest{}, diags
	}
	for key, value := range labelsMap {
		labels = append(labels, cleura.KeyValuePair{
			Key:   key,
			Value: value,
		})
	}

	taintList := listValueToTaintList(ctx, workerGroup.Taints, &diags)
	if diags.HasError() {
		return cleura.WorkerRequest{}, diags
	}

	taints := make([]cleura.Taint, 0)
	for _, t := range taintList {
		taints = append(taints, cleura.Taint{
			Key:    t.Key.ValueString(),
			Value:  t.Value.ValueString(),
			Effect: t.Effect.ValueString(),
		})
	}

	zones := listStringToStringSlice(workerGroup.Zones, &diags)
	if diags.HasError() {
		return cleura.WorkerRequest{}, diags
	}

	request = cleura.WorkerRequest{
		Name:    workerGroup.WorkerGroupName.ValueString(),
		Minimum: minNodes,
		Maximum: maxNodes,
		Machine: cleura.MachineDetails{
			Type: workerGroup.MachineType.ValueString(),
			Image: cleura.ImageDetails{
				Name:    workerGroup.ImageName.ValueString(),
				Version: workerGroup.ImageVersion.ValueString(),
			},
		},
		Volume: cleura.VolumeDetails{
			Size: workerGroup.VolumeSize.ValueString(),
		},
		Annotations: annotations,
		Labels:      labels,
		Taints:      taints,
		Zones:       zones,
	}

	return request, diags
}

func cleuraWorkerToObjectValue(ctx context.Context, worker cleura.WorkerUpdateResponse) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	annotations, err := types.MapValueFrom(ctx, types.StringType, worker.Annotations)
	diags.Append(err...)

	labels, err := types.MapValueFrom(ctx, types.StringType, worker.Labels)
	diags.Append(err...)

	taints, err := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: taintAttrTypesV0()}, cleuraTaintListToTaintList(worker.Taints))
	diags.Append(err...)

	zones, err := types.ListValueFrom(ctx, types.StringType, worker.Zones)
	diags.Append(err...)

	objVal, err := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), workerGroupModelV1{
		WorkerGroupName: types.StringValue(worker.Name),
		MachineType:     types.StringValue(worker.Machine.Type),
		ImageName:       types.StringValue(worker.Machine.Image.Name),
		ImageVersion:    types.StringValue(worker.Machine.Image.Version),
		VolumeSize:      types.StringValue(worker.Volume.Size),
		MinNodes:        types.Int64Value(int64(worker.Minimum)),
		MaxNodes:        types.Int64Value(int64(worker.Maximum)),
		Annotations:     annotations,
		Labels:          labels,
		Taints:          taints,
		Zones:           zones,
	})
	diags.Append(err...)

	return objVal, diags
}

func cleuraWorkerCreateToObjectValue(ctx context.Context, worker cleura.WorkerCreateResponse) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	var annotationsMap map[string]string
	if len(worker.Annotations) > 0 {
		annotationsMap = make(map[string]string)
		for _, annotation := range worker.Annotations {
			annotationsMap[annotation.Key] = annotation.Value
		}
	}
	annotations, err := types.MapValueFrom(ctx, types.StringType, annotationsMap)
	diags.Append(err...)

	var labelsMap map[string]string
	if len(worker.Labels) > 0 {
		labelsMap = make(map[string]string)
		for _, label := range worker.Labels {
			labelsMap[label.Key] = label.Value
		}
	}
	labels, err := types.MapValueFrom(ctx, types.StringType, labelsMap)
	diags.Append(err...)

	taints, err := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: taintAttrTypesV0()}, cleuraTaintListToTaintList(worker.Taints))
	diags.Append(err...)

	zones, err := types.ListValueFrom(ctx, types.StringType, worker.Zones)
	diags.Append(err...)

	objVal, err := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), workerGroupModelV1{
		WorkerGroupName: types.StringValue(worker.Name),
		MachineType:     types.StringValue(worker.Machine.Type),
		ImageName:       types.StringValue(worker.Machine.Image.Name),
		ImageVersion:    types.StringValue(worker.Machine.Image.Version),
		VolumeSize:      types.StringValue(worker.Volume.Size),
		MinNodes:        types.Int64Value(int64(worker.Minimum)),
		MaxNodes:        types.Int64Value(int64(worker.Maximum)),
		Annotations:     annotations,
		Labels:          labels,
		Taints:          taints,
		Zones:           zones,
	})
	diags.Append(err...)

	return objVal, diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *shootClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "XXX_CREATE")
	var plan shootClusterResourceModelV1
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

	var workerGroups []attr.Value
	for _, group := range plan.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		workerGroups = append(workerGroups, objVal)
	}

	// Mapping defined workers
	var clusterWorkers []cleura.WorkerRequest

	for _, wg := range workerGroups {
		worker := attrValuesToWorkerGroupModelV1(wg, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		workerGroupRequest, diags := createWorkerRequestV1(ctx, worker)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		clusterWorkers = append(clusterWorkers, workerGroupRequest)
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

	maintenance := attrValuesToMaintenanceModelV0(plan.Maintenance, &resp.Diagnostics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	network := &cleura.WorkerNetwork{}
	if plan.ProviderDetails.NetworkId.ValueString() != "" && plan.ProviderDetails.RouterId.ValueString() != "" {
		network = &cleura.WorkerNetwork{
			Id: plan.ProviderDetails.NetworkId.ValueString(),
			Router: cleura.Router{
				Id: plan.ProviderDetails.RouterId.ValueString(),
			},
		}
	}

	if plan.ProviderDetails.WorkerCidr.ValueString() != "" {
		network.WorkersCIDR = plan.ProviderDetails.WorkerCidr.ValueString()
	}

	//------------------------------
	clusterRequest := cleura.ShootClusterRequest{
		Shoot: cleura.ShootClusterRequestConfig{
			Name: plan.Name.ValueString(),
			KubernetesVersion: &cleura.K8sVersion{
				Version: plan.K8sVersion.ValueString(),
			},
			Provider: &cleura.ProviderDetailsRequest{
				InfrastructureConfig: cleura.InfrastructureConfigDetails{
					FloatingPoolName: plan.ProviderDetails.FloatingPoolName.ValueString(),
					Networks:         network,
				},
				Workers: clusterWorkers,
			},
			Maintenance: &cleura.MaintenanceDetails{
				AutoUpdate: &cleura.AutoUpdateDetails{
					KubernetesVersion:   maintenance.AutoUpdateKubernetes.ValueBool(),
					MachineImageVersion: maintenance.AutoUpdateMachineImage.ValueBool(),
				},
				TimeWindow: &cleura.TimeWindowDetails{
					Begin: maintenance.TimeWindowBegin.ValueString(),
					End:   maintenance.TimeWindowEnd.ValueString(),
				},
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

	shootResponse, err := r.client.CreateShootCluster(plan.GardenerDomain.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), clusterRequest)
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

	// Set computed network field
	plan.ProviderDetails.NetworkId = types.StringValue(shootResponse.Shoot.Provider.InfrastructureConfig.Networks.Id)
	plan.ProviderDetails.RouterId = types.StringValue(shootResponse.Shoot.Provider.InfrastructureConfig.Networks.Router.Id)
	plan.ProviderDetails.WorkerCidr = types.StringValue(shootResponse.Shoot.Provider.InfrastructureConfig.Networks.WorkersCIDR)

	// Reset the current worker groups value and store the computed values
	workerGroups = make([]attr.Value, 0)
	for _, worker := range shootResponse.Shoot.Provider.Workers {
		obj, diags := cleuraWorkerCreateToObjectValue(ctx, worker)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		workerGroups = append(workerGroups, obj)
	}

	plan.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err = clusterReadyOperationWaiter(r.client, ctx, createTimeout, plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
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

func clusterReconcileWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, gardenerDomain string, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.InitialInterval = 120 * time.Second
	b.MaxInterval = 75 * time.Second
	b.Multiplier = 2
	operation := func() error {
		clusterResp, err := client.GetShootCluster(gardenerDomain, clusterName, clusterRegion, clusterProject)
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

func clusterReadyOperationWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, gardenerDomain string, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.MaxInterval = 75 * time.Second
	b.InitialInterval = 120 * time.Second
	b.Multiplier = 2
	operation := func() error {
		clusterResp, err := client.GetShootCluster(gardenerDomain, clusterName, clusterRegion, clusterProject)
		if err != nil {
			return backoff.Permanent(err)
		}
		if len(clusterResp.Status.Conditions) < 1 {
			return errors.New("cluster has no events yet")
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

func deleteClusterOperationWaiter(client *cleura.Client, ctx context.Context, maxRetryTime time.Duration, gardenerDomain string, clusterName string, clusterRegion string, clusterProject string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime - 1*time.Minute
	b.MaxInterval = 75 * time.Second
	b.InitialInterval = 120 * time.Second
	b.Multiplier = 2
	operation := func() error {

		_, err := client.GetShootCluster(gardenerDomain, clusterName, clusterRegion, clusterProject)
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
	var state shootClusterResourceModelV1
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed shoot cluster from cleura
	shootResponse, err := r.client.GetShootCluster(state.GardenerDomain.ValueString(), state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())
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

	// Set computed network field
	state.ProviderDetails.NetworkId = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.Networks.Id)
	state.ProviderDetails.RouterId = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.Networks.Router.Id)
	state.ProviderDetails.WorkerCidr = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.Networks.WorkersCIDR)

	var workerGroups []attr.Value
	for _, group := range state.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		workerGroups = append(workerGroups, objVal)
	}

	state.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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

	// Write existing maintenance configuration to state
	state.Maintenance, diags = types.ObjectValueFrom(ctx, maintenanceAttrTypesV0(), maintenanceModel{
		AutoUpdateKubernetes:   types.BoolValue(shootResponse.Spec.Maintenance.AutoUpdate.KubernetesVersion),
		AutoUpdateMachineImage: types.BoolValue(shootResponse.Spec.Maintenance.AutoUpdate.MachineImageVersion),
		TimeWindowBegin:        types.StringValue(shootResponse.Spec.Maintenance.TimeWindow.Begin),
		TimeWindowEnd:          types.StringValue(shootResponse.Spec.Maintenance.TimeWindow.End),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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
	var plan shootClusterResourceModelV1
	var currentState shootClusterResourceModelV1
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

	if !reflect.DeepEqual(plan.HibernationSchedules, currentState.HibernationSchedules) || !plan.Maintenance.Equal(currentState.Maintenance) || !reflect.DeepEqual(plan.K8sVersion, currentState.K8sVersion) {
		tflog.Debug(ctx, "Hibernation schedules or K8s version changed")

		hibernationSchedules := []cleura.HibernationSchedule{}
		for _, schedule := range plan.HibernationSchedules {
			hibernationSchedules = append(hibernationSchedules, cleura.HibernationSchedule{
				Start: schedule.Start.ValueString(),
				End:   schedule.End.ValueString(),
			},
			)
		}

		maintenance := attrValuesToMaintenanceModelV0(plan.Maintenance, &resp.Diagnostics)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		clusterUpdateRequest := cleura.ShootClusterRequest{
			Shoot: cleura.ShootClusterRequestConfig{
				KubernetesVersion: &cleura.K8sVersion{
					Version: plan.K8sVersion.ValueString(),
				},
				Hibernation: &cleura.HibernationSchedules{
					HibernationSchedules: hibernationSchedules,
				},
				Maintenance: &cleura.MaintenanceDetails{
					AutoUpdate: &cleura.AutoUpdateDetails{
						KubernetesVersion:   maintenance.AutoUpdateKubernetes.ValueBool(),
						MachineImageVersion: maintenance.AutoUpdateMachineImage.ValueBool(),
					},
					TimeWindow: &cleura.TimeWindowDetails{
						Begin: maintenance.TimeWindowBegin.ValueString(),
						End:   maintenance.TimeWindowEnd.ValueString(),
					},
				},
			},
		}

		_, err := r.client.UpdateShootCluster(plan.GardenerDomain.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), plan.Name.ValueString(), clusterUpdateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating shoot cluster",
				"Could not update cluster, unexpected error: "+err.Error(),
			)
			return
		}
		err = clusterReconcileWaiter(r.client, ctx, createTimeout, plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error while waiting for cluster to become ready (modify)",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

	}

	tflog.Debug(ctx, "Workergroups changed")

	var plannedWorkerGroups []attr.Value
	for _, group := range plan.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		plannedWorkerGroups = append(plannedWorkerGroups, objVal)
	}

	var currentWorkerGroups []attr.Value
	for _, group := range currentState.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		currentWorkerGroups = append(currentWorkerGroups, objVal)
	}

	wgModify, wgCreate, wgDelete := getCreateModifyDeleteWorkgroups(
		attrValuesToWorkerGroupModelSlice(plannedWorkerGroups, &resp.Diagnostics),
		attrValuesToWorkerGroupModelSlice(currentWorkerGroups, &resp.Diagnostics),
	)

	tflog.Debug(ctx, fmt.Sprintf("modify: %+v, create: %+v, delete: %+v, plan: %+v, state: %+v", wgModify, wgCreate, wgDelete, plan.ProviderDetails.WorkerGroups, currentState.ProviderDetails.WorkerGroups))
	for _, wg := range wgModify {
		// Create a request for the workergroup to modify
		workerGroupRequest, diags := createWorkerRequestV1(ctx, wg)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		_, err := r.client.UpdateWorkerGroup(plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), wg.WorkerGroupName.ValueString(), cleura.WorkerGroupRequest{Worker: workerGroupRequest})
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error Updating Worker Group",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

		err = clusterReconcileWaiter(r.client, ctx, createTimeout, plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error while waiting for cluster to become ready (modify)",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}
	}
	for _, wg := range wgCreate {
		workerGroupRequest, diags := createWorkerRequestV1(ctx, wg)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		_, err := r.client.AddWorkerGroup(plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), cleura.WorkerGroupRequest{Worker: workerGroupRequest})
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error Adding Worker Group",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

		err = clusterReconcileWaiter(r.client, ctx, createTimeout, plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error while waiting for cluster to become ready (modify)",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

	}
	for _, wg := range wgDelete {
		_, err := r.client.DeleteWorkerGroup(plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString(), wg.WorkerGroupName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error Deleting Worker Group",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}

		err = clusterReconcileWaiter(r.client, ctx, createTimeout, plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error while waiting for cluster to become ready (modify)",
				fmt.Sprintf("... details ... %s", err),
			)
			return
		}
	}

	clusterUpdateResp, err := r.client.GetShootCluster(plan.GardenerDomain.ValueString(), plan.Name.ValueString(), plan.Region.ValueString(), plan.Project.ValueString())
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

	var workerGroups []attr.Value

	for _, worker := range clusterUpdateResp.Spec.Provider.Workers {
		obj, diags := cleuraWorkerToObjectValue(ctx, worker)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		workerGroups = append(workerGroups, obj)
	}

	plan.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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
	var state shootClusterResourceModelV1
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
	_, err := r.client.DeleteShootCluster(state.GardenerDomain.ValueString(), state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Shoot Cluster",
			"Could not delete Shoot Cluster, unexpected error: "+err.Error(),
		)
		return
	}
	// Wait until API responds with 404
	err = deleteClusterOperationWaiter(r.client, ctx, createTimeout, state.GardenerDomain.ValueString(), state.Name.ValueString(), state.Region.ValueString(), state.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(

			"API Error Shoot Cluster Resource status check",
			fmt.Sprintf("... details ... %s", err),
		)
		return
	}
}

func getCreateModifyDeleteWorkgroups(wgsPlan []workerGroupModelV1, wgsState []workerGroupModelV1) (wgModify []workerGroupModelV1, wgCreate []workerGroupModelV1, wgDelete []workerGroupModelV1) {
	stateMap := make(map[string]workerGroupModelV1)
	for i, wg := range wgsState {
		stateMap[wg.WorkerGroupName.ValueString()] = wgsState[i]
	}
	planMap := make(map[string]workerGroupModelV1)
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
	var state shootClusterResourceModelV1
	tflog.Debug(ctx, fmt.Sprintf("idparts: %v", idParts))
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: GardenerDomain,Name,Region,Project_id. Got: %q", req.ID),
		)
		return
	}
	state.GardenerDomain = types.StringValue(idParts[0])
	state.Name = types.StringValue(idParts[1])
	state.Region = types.StringValue(idParts[2])
	state.Project = types.StringValue(idParts[3])

	// Get refreshed shoot cluster from cleura
	shootResponse, err := r.client.GetShootCluster(idParts[0], idParts[1], idParts[2], idParts[3])
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
	state.ProviderDetails.NetworkId = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.Networks.Id)
	state.ProviderDetails.RouterId = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.Networks.Router.Id)
	state.ProviderDetails.WorkerCidr = types.StringValue(shootResponse.Spec.Provider.InfrastructureConfig.Networks.WorkersCIDR)

	// make an attr.Value slice and fill with existing workers
	var workerGroups []attr.Value
	for _, group := range state.ProviderDetails.WorkerGroups.Elements() {
		objVal, diags := types.ObjectValueFrom(ctx, workerGroupModelAttrTypesV1(), group)
		resp.Diagnostics.Append(diags...)
		workerGroups = append(workerGroups, objVal)
	}

	// add imported worker groups to state
	for _, worker := range shootResponse.Spec.Provider.Workers {
		obj, diags := cleuraWorkerToObjectValue(ctx, worker)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		workerGroups = append(workerGroups, obj)
	}

	var diags diag.Diagnostics

	// Write existing maintenance configuration to state
	state.Maintenance, diags = types.ObjectValueFrom(ctx, maintenanceAttrTypesV0(), maintenanceModel{
		AutoUpdateKubernetes:   types.BoolValue(shootResponse.Spec.Maintenance.AutoUpdate.KubernetesVersion),
		AutoUpdateMachineImage: types.BoolValue(shootResponse.Spec.Maintenance.AutoUpdate.MachineImageVersion),
		TimeWindowBegin:        types.StringValue(shootResponse.Spec.Maintenance.TimeWindow.Begin),
		TimeWindowEnd:          types.StringValue(shootResponse.Spec.Maintenance.TimeWindow.End),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ProviderDetails.WorkerGroups, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workerGroupModelAttrTypesV1()}, workerGroups)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
