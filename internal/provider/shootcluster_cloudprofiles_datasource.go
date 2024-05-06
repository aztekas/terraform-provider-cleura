package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/aztekas/cleura-client-go/pkg/api/cleura"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ datasource.DataSource              = &shootClusterProfilesDataSource{}
	_ datasource.DataSourceWithConfigure = &shootClusterProfilesDataSource{}
)

type machineTypesFilter struct {
	Cpu    types.String `tfsdk:"cpu"`
	Memory types.String `tfsdk:"memory"`
}
type kubernetesFilter struct {
	Supported types.Bool `tfsdk:"supported_only"`
}
type machineImageFilter struct {
	Supported types.Bool `tfsdk:"supported_only"`
}

type shootClusterProfileFilters struct {
	MachineTypeFilter  *machineTypesFilter `tfsdk:"machine_types"`
	KubernetesFilter   *kubernetesFilter   `tfsdk:"kubernetes"`
	MachineImageFilter *machineImageFilter `tfsdk:"machine_images"`
}

type shootClusterProfilesDataSourceModel struct {
	KubernetesLatest   types.String                            `tfsdk:"kubernetes_latest"`
	MachineImageLatest types.String                            `tfsdk:"gardenlinux_image_latest"`
	Filters            *shootClusterProfileFilters             `tfsdk:"filters"`
	KubernetesVersions []shootClusterProfilesVersionModel      `tfsdk:"kubernetes_versions"`
	MachineImages      []shootClusterProfilesMachineImageModel `tfsdk:"machine_images"`
	MachineTypes       []shootClusterProfilesMachineTypeModel  `tfsdk:"machine_types"`
}

type shootClusterProfilesVersionModel struct {
	Version        types.String `tfsdk:"version"`
	Classification types.String `tfsdk:"classification"`
	Expires        types.String `tfsdk:"expiration_date"`
}

type shootClusterProfilesMachineImageModel struct {
	Name     types.String                       `tfsdk:"name"`
	Versions []shootClusterProfilesVersionModel `tfsdk:"versions"`
}

type shootClusterProfilesMachineTypeModel struct {
	Cpu          types.String `tfsdk:"cpu"`
	Gpu          types.String `tfsdk:"gpu"`
	Memory       types.String `tfsdk:"memory"`
	Name         types.String `tfsdk:"name"`
	Usable       types.Bool   `tfsdk:"usable"`
	Architecture types.String `tfsdk:"architecture"`
}

func NewShootClusterProfilesDataSource() datasource.DataSource {
	return &shootClusterProfilesDataSource{}
}

type shootClusterProfilesDataSource struct {
	client *cleura.Client
}

// Metadata returns the data source type name.
func (d *shootClusterProfilesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shoot_cluster_profiles"
}

// Schema defines the schema for the data source.
func (d *shootClusterProfilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"kubernetes_latest": schema.StringAttribute{
				Computed: true,
				Required: false,
			},
			"gardenlinux_image_latest": schema.StringAttribute{
				Computed: true,
				Required: false,
			},
			"filters": schema.SingleNestedAttribute{
				Computed:    false,
				Required:    false,
				Description: "Filter output profile",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"machine_types": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"cpu": schema.StringAttribute{
								Optional: true,
								Computed: false,
								Validators: []validator.String{
									stringvalidator.AtLeastOneOf(path.Expressions{
										path.MatchRoot("filters").AtName("machine_types").AtName("memory"),
										path.MatchRoot("filters").AtName("machine_types").AtName("cpu"),
									}...),
								},
							},
							"memory": schema.StringAttribute{
								Optional: true,
								Computed: false,
							},
						},
						Optional: true,
						Computed: false,
						Validators: []validator.Object{
							objectvalidator.AtLeastOneOf(path.Expressions{
								path.MatchRoot("filters").AtName("machine_types"),
								path.MatchRoot("filters").AtName("machine_images"),
								path.MatchRoot("filters").AtName("kubernetes"),
							}...),
						},
					},
					"kubernetes": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"supported_only": schema.BoolAttribute{
								Optional: true,
								Computed: false,
								Validators: []validator.Bool{
									boolvalidator.AtLeastOneOf(path.Expressions{
										path.MatchRoot("filters").AtName("kubernetes").AtName("supported_only"),
									}...),
								},
							},
						},
						Optional: true,
						Computed: false,
					},
					"machine_images": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"supported_only": schema.BoolAttribute{
								Optional: true,
								Computed: false,
								Validators: []validator.Bool{
									boolvalidator.AtLeastOneOf(path.Expressions{
										path.MatchRoot("filters").AtName("machine_images").AtName("supported_only"),
									}...),
								},
							},
						},
						Optional: true,
						Computed: false,
					},
				},
			},
			"kubernetes_versions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Available Kubernetes versions",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							Computed: true,
						},
						"classification": schema.StringAttribute{
							Computed: true,
						},
						"expiration_date": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"machine_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Available machine types",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"cpu": schema.StringAttribute{
							Computed: true,
						},
						"memory": schema.StringAttribute{
							Computed: true,
						},
						"gpu": schema.StringAttribute{
							Computed: true,
						},
						"usable": schema.BoolAttribute{
							Computed: true,
						},
						"architecture": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"machine_images": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Available machine images",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"versions": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Version details",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"version": schema.StringAttribute{
										Computed: true,
									},
									"classification": schema.StringAttribute{
										Computed: true,
									},
									"expiration_date": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *shootClusterProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state shootClusterProfilesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, err := d.client.GetCloudProfile()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get profile data",
			err.Error(),
		)
		return
	}
	state.KubernetesLatest = getLatestK8sVersion(profile)
	state.MachineImageLatest = getLatestGardenlinuxVersion(profile)
	fProfile := filterProfile(profile, state.Filters)

	for _, version := range fProfile.Spec.Kubernetes.Versions {
		state.KubernetesVersions = append(state.KubernetesVersions, shootClusterProfilesVersionModel{
			Version:        types.StringValue(version.Version),
			Classification: types.StringValue(version.Classification),
			Expires:        types.StringValue(version.ExpirationDate),
		})
	}
	for _, machineType := range fProfile.Spec.MachineTypes {
		state.MachineTypes = append(state.MachineTypes, shootClusterProfilesMachineTypeModel{
			Name:         types.StringValue(machineType.Name),
			Cpu:          types.StringValue(machineType.Cpu),
			Memory:       types.StringValue(machineType.Memory),
			Gpu:          types.StringValue(machineType.Gpu),
			Architecture: types.StringValue(machineType.Architecture),
			Usable:       types.BoolValue(machineType.Usable),
		})

	}
	var imageVersions []shootClusterProfilesVersionModel
	for _, machineImage := range fProfile.Spec.MachineImages {
		for _, machineImageVersion := range machineImage.Versions {

			imageVersions = append(imageVersions, shootClusterProfilesVersionModel{
				Version:        types.StringValue(machineImageVersion.Version),
				Classification: types.StringValue(machineImageVersion.Classification),
				Expires:        types.StringValue(machineImageVersion.ExpirationDate),
			})
		}
		state.MachineImages = append(state.MachineImages, shootClusterProfilesMachineImageModel{
			Name:     types.StringValue(machineImage.Name),
			Versions: imageVersions,
		})
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *shootClusterProfilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func filterProfile(p *cleura.CloudProfile, f *shootClusterProfileFilters) *cleura.CloudProfile {
	if f == nil {
		return p
	}
	if f.MachineImageFilter != nil {
		if f.MachineImageFilter.Supported.ValueBool() {
			for i, mi := range p.Spec.MachineImages {
				var fVersions []cleura.CPVersion
				for _, miVersion := range mi.Versions {
					if miVersion.Classification == "supported" {
						fVersions = append(fVersions, miVersion)
					}
				}
				p.Spec.MachineImages[i].Versions = fVersions
			}
		}
	}
	if f.KubernetesFilter != nil {
		if f.KubernetesFilter.Supported.ValueBool() {
			var fVersions []cleura.CPVersion
			for _, kv := range p.Spec.Kubernetes.Versions {
				if kv.Classification == "supported" {
					fVersions = append(fVersions, kv)
				}
			}
			p.Spec.Kubernetes.Versions = fVersions
		}
	}
	if f.MachineTypeFilter != nil {
		memFilter := f.MachineTypeFilter.Memory.ValueString() != ""
		cpuFilter := f.MachineTypeFilter.Cpu.ValueString() != ""

		var fTypes []cleura.CPMachineType
		for _, mt := range p.Spec.MachineTypes {
			if memFilter && cpuFilter {
				if mt.Cpu == f.MachineTypeFilter.Cpu.ValueString() && mt.Memory == f.MachineTypeFilter.Memory.ValueString() {
					fTypes = append(fTypes, mt)
				}
			}
			if memFilter && !cpuFilter {
				if mt.Memory == f.MachineTypeFilter.Memory.ValueString() {
					fTypes = append(fTypes, mt)
				}
			}
			if cpuFilter && !memFilter {
				if mt.Cpu == f.MachineTypeFilter.Cpu.ValueString() {
					fTypes = append(fTypes, mt)
				}
			}
		}
		p.Spec.MachineTypes = fTypes
	}
	return p
}

func getLatestK8sVersion(p *cleura.CloudProfile) basetypes.StringValue {
	var listK8sVersions []string
	for _, kVer := range p.Spec.Kubernetes.Versions {
		if kVer.Classification == "supported" {
			listK8sVersions = append(listK8sVersions, kVer.Version)
		}
	}
	vers := make([]*version.Version, len(listK8sVersions))
	for i, raw := range listK8sVersions {
		v, _ := version.NewVersion(raw)
		vers[i] = v
	}
	sort.Sort(sort.Reverse(version.Collection(vers)))

	return types.StringValue(vers[0].Original())
}

func getLatestGardenlinuxVersion(p *cleura.CloudProfile) basetypes.StringValue {
	var listMachineVersions []string
	for _, mVer := range p.Spec.MachineImages {
		if mVer.Name == "gardenlinux" {
			for _, v := range mVer.Versions {
				if v.Classification == "supported" {
					listMachineVersions = append(listMachineVersions, v.Version)
				}
			}
		}
	}
	vers := make([]*version.Version, len(listMachineVersions))
	for i, raw := range listMachineVersions {
		v, _ := version.NewVersion(raw)
		vers[i] = v
	}
	sort.Sort(sort.Reverse(version.Collection(vers)))
	return types.StringValue(vers[0].Original())
}
