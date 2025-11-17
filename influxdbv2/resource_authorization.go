package influxdbv2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AuthorizationResource{}
var _ resource.ResourceWithImportState = &AuthorizationResource{}

func NewAuthorizationResource() resource.Resource {
	return &AuthorizationResource{}
}

// AuthorizationResource defines the resource implementation.
type AuthorizationResource struct {
	client influxdb2.Client
}

// AuthorizationResourceModel describes the resource data model.
type AuthorizationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	Permissions types.Set    `tfsdk:"permissions"`
	UserID      types.String `tfsdk:"user_id"`
	UserOrgID   types.String `tfsdk:"user_org_id"`
	Token       types.String `tfsdk:"token"`
}

// PermissionModel describes the permission data model.
type PermissionModel struct {
	Action   types.String `tfsdk:"action"`
	Resource types.Set    `tfsdk:"resource"`
}

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Org   types.String `tfsdk:"org"`
	OrgID types.String `tfsdk:"org_id"`
	Type  types.String `tfsdk:"type"`
}

func (r *AuthorizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization"
}

func (r *AuthorizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an InfluxDB v2 authorization (API token).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the authorization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the authorization.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"status": schema.StringAttribute{
				Description: "Status of the authorization. Valid values are 'active' or 'inactive'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("active"),
			},
			"permissions": schema.SetNestedAttribute{
				Description:   "List of permissions for the authorization.",
				Required:      true,
				PlanModifiers: []planmodifier.Set{
					// Permissions cannot be updated once created
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: "Permission action (e.g., 'read', 'write').",
							Required:    true,
						},
						"resource": schema.SetNestedAttribute{
							Description: "Resources the permission applies to.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: "Resource ID.",
										Required:    true,
									},
									"org": schema.StringAttribute{
										Description: "Organization name.",
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString(""),
									},
									"org_id": schema.StringAttribute{
										Description: "Organization ID.",
										Required:    true,
									},
									"type": schema.StringAttribute{
										Description: "Resource type (e.g., 'buckets', 'dashboards').",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The user ID associated with the authorization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_org_id": schema.StringAttribute{
				Description: "The organization ID of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				Description: "The authorization token. This is sensitive and should be stored securely.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AuthorizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(influxdb2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected influxdb2.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *AuthorizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthorizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert permissions from Terraform data to domain model
	permissions, err := r.convertPermissionsToDomain(ctx, plan.Permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Permissions",
			"Could not convert permissions: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Creating authorization", map[string]any{"permissions_count": len(permissions)})

	// Create authorization
	orgID := plan.OrgID.ValueString()
	desc := plan.Description.ValueString()
	status := domain.AuthorizationUpdateRequestStatus(plan.Status.ValueString())

	authorization := domain.Authorization{
		AuthorizationUpdateRequest: domain.AuthorizationUpdateRequest{
			Description: &desc,
			Status:      &status,
		},
		OrgID:       &orgID,
		Permissions: &permissions,
	}

	result, err := r.client.AuthorizationsAPI().CreateAuthorization(ctx, &authorization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Authorization",
			"Could not create authorization: "+err.Error(),
		)
		return
	}

	// Set the ID and computed fields
	plan.ID = types.StringValue(*result.Id)
	if result.Token != nil {
		plan.Token = types.StringValue(*result.Token)
	}
	if result.UserID != nil {
		plan.UserID = types.StringValue(*result.UserID)
	}
	if result.OrgID != nil {
		plan.UserOrgID = types.StringValue(*result.OrgID)
	}

	tflog.Trace(ctx, "Created authorization", map[string]any{"id": plan.ID.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthorizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the authorization from InfluxDB
	if err := r.readAuthorization(ctx, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Authorization",
			"Could not read authorization ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthorizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Note: Only status can be updated in InfluxDB authorizations
	id := plan.ID.ValueString()
	authorization := domain.Authorization{
		Id: &id,
	}
	statusUpdate := domain.AuthorizationUpdateRequestStatus(plan.Status.ValueString())

	tflog.Debug(ctx, "Updating authorization status", map[string]any{"id": plan.ID.ValueString(), "status": plan.Status.ValueString()})

	_, err := r.client.AuthorizationsAPI().UpdateAuthorizationStatus(ctx, &authorization, statusUpdate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Authorization",
			"Could not update authorization status: "+err.Error(),
		)
		return
	}

	// Read the updated authorization
	if err := r.readAuthorization(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Authorization After Update",
			"Could not read authorization after update: "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthorizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting authorization", map[string]any{"id": state.ID.ValueString()})

	id := state.ID.ValueString()
	authorization := domain.Authorization{
		Id: &id,
	}

	err := r.client.AuthorizationsAPI().DeleteAuthorization(ctx, &authorization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Authorization",
			"Could not delete authorization: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "Deleted authorization", map[string]any{"id": state.ID.ValueString()})
}

func (r *AuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to read authorization and populate the model
func (r *AuthorizationResource) readAuthorization(ctx context.Context, model *AuthorizationResourceModel) error {
	// Find all authorizations for the org
	authorizations, err := r.client.AuthorizationsAPI().FindAuthorizationsByOrgID(ctx, model.OrgID.ValueString())
	if err != nil {
		return fmt.Errorf("error finding authorizations: %w", err)
	}

	// Find the specific authorization by ID
	var auth *domain.Authorization
	for i := range *authorizations {
		if *(*authorizations)[i].Id == model.ID.ValueString() {
			auth = &(*authorizations)[i]
			break
		}
	}

	if auth == nil {
		return fmt.Errorf("authorization not found")
	}

	// Update model with data from InfluxDB
	if auth.Status != nil {
		model.Status = types.StringValue(string(*auth.Status))
	}

	if auth.UserID != nil {
		model.UserID = types.StringValue(*auth.UserID)
	}

	if auth.OrgID != nil {
		model.UserOrgID = types.StringValue(*auth.OrgID)
	}

	if auth.Token != nil {
		model.Token = types.StringValue(*auth.Token)
	}

	// Note: Permissions are not returned by the read API, so we keep the plan values

	return nil
}

// Helper function to convert permissions from Terraform Set to domain model
func (r *AuthorizationResource) convertPermissionsToDomain(ctx context.Context, permsSet types.Set) ([]domain.Permission, error) {
	var permissions []PermissionModel
	diags := permsSet.ElementsAs(ctx, &permissions, false)
	if diags.HasError() {
		return nil, fmt.Errorf("error converting permissions set")
	}

	domainPermissions := []domain.Permission{}
	for _, perm := range permissions {
		var resources []ResourceModel
		diags := perm.Resource.ElementsAs(ctx, &resources, false)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting resources set")
		}

		for _, res := range resources {
			id := res.ID.ValueString()
			orgID := res.OrgID.ValueString()
			org := res.Org.ValueString()
			name := ""

			domainResource := domain.Resource{
				Type:  domain.ResourceType(res.Type.ValueString()),
				Id:    &id,
				OrgID: &orgID,
				Name:  &name,
				Org:   &org,
			}

			domainPerm := domain.Permission{
				Action:   domain.PermissionAction(perm.Action.ValueString()),
				Resource: domainResource,
			}

			domainPermissions = append(domainPermissions, domainPerm)
		}
	}

	return domainPermissions, nil
}

// Helper function to convert permissions from domain model to Terraform Set
func (r *AuthorizationResource) convertPermissionsToTerraform(ctx context.Context, domainPerms []domain.Permission) (types.Set, error) {
	resourceType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":     types.StringType,
			"org":    types.StringType,
			"org_id": types.StringType,
			"type":   types.StringType,
		},
	}

	permissionType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"action":   types.StringType,
			"resource": types.SetType{ElemType: resourceType},
		},
	}

	elements := []attr.Value{}
	for _, perm := range domainPerms {
		resourceElements := []attr.Value{}

		id := ""
		if perm.Resource.Id != nil {
			id = *perm.Resource.Id
		}
		orgID := ""
		if perm.Resource.OrgID != nil {
			orgID = *perm.Resource.OrgID
		}
		org := ""
		if perm.Resource.Org != nil {
			org = *perm.Resource.Org
		}

		resObj, diags := types.ObjectValue(
			resourceType.AttrTypes,
			map[string]attr.Value{
				"id":     types.StringValue(id),
				"org":    types.StringValue(org),
				"org_id": types.StringValue(orgID),
				"type":   types.StringValue(string(perm.Resource.Type)),
			},
		)
		if diags.HasError() {
			return types.SetNull(permissionType), fmt.Errorf("error creating resource object")
		}
		resourceElements = append(resourceElements, resObj)

		resourceSet, diags := types.SetValue(resourceType, resourceElements)
		if diags.HasError() {
			return types.SetNull(permissionType), fmt.Errorf("error creating resource set")
		}

		permObj, diags := types.ObjectValue(
			permissionType.AttrTypes,
			map[string]attr.Value{
				"action":   types.StringValue(string(perm.Action)),
				"resource": resourceSet,
			},
		)
		if diags.HasError() {
			return types.SetNull(permissionType), fmt.Errorf("error creating permission object")
		}
		elements = append(elements, permObj)
	}

	setValue, diags := types.SetValue(permissionType, elements)
	if diags.HasError() {
		return types.SetNull(permissionType), fmt.Errorf("error creating permissions set")
	}

	return setValue, nil
}
