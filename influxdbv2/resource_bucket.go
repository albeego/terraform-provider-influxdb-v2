package influxdbv2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketResource{}
var _ resource.ResourceWithImportState = &BucketResource{}

func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

// BucketResource defines the resource implementation.
type BucketResource struct {
	client influxdb2.Client
}

// BucketResourceModel describes the resource data model.
type BucketResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrgID          types.String `tfsdk:"org_id"`
	RetentionRules types.Set    `tfsdk:"retention_rules"`
	RP             types.String `tfsdk:"rp"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	Type           types.String `tfsdk:"type"`
}

// RetentionRuleModel describes the retention rule data model.
type RetentionRuleModel struct {
	EverySeconds types.Int64  `tfsdk:"every_seconds"`
	Type         types.String `tfsdk:"type"`
}

func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an InfluxDB v2 bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the bucket.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the bucket.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the bucket.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"retention_rules": schema.SetNestedAttribute{
				Description: "Retention rules for the bucket.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"every_seconds": schema.Int64Attribute{
							Description: "Duration in seconds for how long data will be kept in the database.",
							Required:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"type": schema.StringAttribute{
							Description: "Type of retention rule. Defaults to 'expire'.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("expire"),
						},
					},
				},
			},
			"rp": schema.StringAttribute{
				Description: "The retention policy name.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the bucket was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the bucket was last updated.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the bucket.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert retention rules from Terraform data to domain model
	retentionRules, err := r.convertRetentionRulesToDomain(ctx, plan.RetentionRules)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Retention Rules",
			"Could not convert retention rules: "+err.Error(),
		)
		return
	}

	// Create bucket
	desc := plan.Description.ValueString()
	orgID := plan.OrgID.ValueString()
	rp := plan.RP.ValueString()

	newBucket := &domain.Bucket{
		Description:    &desc,
		Name:           plan.Name.ValueString(),
		OrgID:          &orgID,
		RetentionRules: retentionRules,
		Rp:             &rp,
	}

	tflog.Debug(ctx, "Creating bucket", map[string]any{"name": plan.Name.ValueString()})

	result, err := r.client.BucketsAPI().CreateBucket(ctx, newBucket)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Bucket",
			"Could not create bucket, unexpected error: "+err.Error(),
		)
		return
	}

	// Set the ID and read the resource to populate computed fields
	plan.ID = types.StringValue(*result.Id)

	// Read the created bucket to get all computed fields
	if err := r.readBucket(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Bucket After Creation",
			"Could not read bucket after creation: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "Created bucket", map[string]any{"id": plan.ID.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the bucket from InfluxDB
	if err := r.readBucket(ctx, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Bucket",
			"Could not read bucket ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert retention rules from Terraform data to domain model
	retentionRules, err := r.convertRetentionRulesToDomain(ctx, plan.RetentionRules)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Retention Rules",
			"Could not convert retention rules: "+err.Error(),
		)
		return
	}

	// Update bucket
	id := plan.ID.ValueString()
	desc := plan.Description.ValueString()
	orgID := plan.OrgID.ValueString()
	rp := plan.RP.ValueString()

	updateBucket := &domain.Bucket{
		Id:             &id,
		Description:    &desc,
		Name:           plan.Name.ValueString(),
		OrgID:          &orgID,
		RetentionRules: retentionRules,
		Rp:             &rp,
	}

	tflog.Debug(ctx, "Updating bucket", map[string]any{"id": plan.ID.ValueString()})

	_, err = r.client.BucketsAPI().UpdateBucket(ctx, updateBucket)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Bucket",
			"Could not update bucket, unexpected error: "+err.Error(),
		)
		return
	}

	// Read the updated bucket to get all current fields
	if err := r.readBucket(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Bucket After Update",
			"Could not read bucket after update: "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket", map[string]any{"id": state.ID.ValueString()})

	// Delete the bucket
	err := r.client.BucketsAPI().DeleteBucketWithID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Bucket",
			"Could not delete bucket, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "Deleted bucket", map[string]any{"id": state.ID.ValueString()})
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to read bucket and populate the model
func (r *BucketResource) readBucket(ctx context.Context, model *BucketResourceModel) error {
	result, err := r.client.BucketsAPI().FindBucketByID(ctx, model.ID.ValueString())
	if err != nil {
		return fmt.Errorf("error finding bucket: %w", err)
	}

	// Update model with data from InfluxDB
	model.Name = types.StringValue(result.Name)

	if result.Description != nil {
		model.Description = types.StringValue(*result.Description)
	} else {
		model.Description = types.StringValue("")
	}

	if result.OrgID != nil {
		model.OrgID = types.StringValue(*result.OrgID)
	}

	if result.Rp != nil {
		model.RP = types.StringValue(*result.Rp)
	} else {
		model.RP = types.StringValue("")
	}

	if result.CreatedAt != nil {
		model.CreatedAt = types.StringValue(result.CreatedAt.String())
	}

	if result.UpdatedAt != nil {
		model.UpdatedAt = types.StringValue(result.UpdatedAt.String())
	}

	if result.Type != nil {
		model.Type = types.StringValue(string(*result.Type))
	}

	// Convert retention rules
	retentionRulesSet, err := r.convertRetentionRulesToTerraform(ctx, result.RetentionRules)
	if err != nil {
		return fmt.Errorf("error converting retention rules: %w", err)
	}
	model.RetentionRules = retentionRulesSet

	return nil
}

// Helper function to convert retention rules from Terraform Set to domain model
func (r *BucketResource) convertRetentionRulesToDomain(ctx context.Context, rulesSet types.Set) (domain.RetentionRules, error) {
	var rules []RetentionRuleModel
	diags := rulesSet.ElementsAs(ctx, &rules, false)
	if diags.HasError() {
		return nil, fmt.Errorf("error converting retention rules set")
	}

	domainRules := domain.RetentionRules{}
	for _, rule := range rules {
		ruleType := domain.RetentionRuleType(rule.Type.ValueString())
		domainRules = append(domainRules, domain.RetentionRule{
			EverySeconds: rule.EverySeconds.ValueInt64(),
			Type:         &ruleType,
		})
	}

	return domainRules, nil
}

// Helper function to convert retention rules from domain model to Terraform Set
func (r *BucketResource) convertRetentionRulesToTerraform(ctx context.Context, domainRules domain.RetentionRules) (types.Set, error) {
	retentionRuleType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"every_seconds": types.Int64Type,
			"type":          types.StringType,
		},
	}

	elements := []attr.Value{}
	for _, rule := range domainRules {
		ruleTypeValue := "expire"
		if rule.Type != nil {
			ruleTypeValue = string(*rule.Type)
		}

		ruleObj, diags := types.ObjectValue(
			retentionRuleType.AttrTypes,
			map[string]attr.Value{
				"every_seconds": types.Int64Value(rule.EverySeconds),
				"type":          types.StringValue(ruleTypeValue),
			},
		)
		if diags.HasError() {
			return types.SetNull(retentionRuleType), fmt.Errorf("error creating retention rule object")
		}
		elements = append(elements, ruleObj)
	}

	setValue, diags := types.SetValue(retentionRuleType, elements)
	if diags.HasError() {
		return types.SetNull(retentionRuleType), fmt.Errorf("error creating retention rules set")
	}

	return setValue, nil
}
