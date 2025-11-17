# Code Upgrade Examples: SDK v2 → Plugin Framework

This document shows side-by-side comparisons of how deprecated patterns were upgraded.

## 1. Provider Definition

### Before (SDK v2)
```go
func Provider() *schema.Provider {
    return &schema.Provider{
        Schema: map[string]*schema.Schema{
            "url": {
                Type:        schema.TypeString,
                Optional:    true,
                DefaultFunc: schema.EnvDefaultFunc("INFLUXDB_V2_URL", "http://localhost:8086"),
            },
        },
        ConfigureFunc: providerConfigure,
    }
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
    influx := influxdb2.NewClient(d.Get("url").(string), d.Get("token").(string))
    log.Printf("[DEBUG] influxdb url %s", d.Get("url").(string))
    _, err := influx.Ready(context.Background())
    return influx, err
}
```

### After (Plugin Framework)
```go
func (p *influxdbProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Terraform provider for managing InfluxDB v2 resources.",
        Attributes: map[string]schema.Attribute{
            "url": schema.StringAttribute{
                Description: "InfluxDB server URL. Can also be set via INFLUXDB_V2_URL environment variable.",
                Optional:    true,
            },
        },
    }
}

func (p *influxdbProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config influxdbProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

    url := os.Getenv("INFLUXDB_V2_URL")
    if !config.URL.IsNull() {
        url = config.URL.ValueString()
    }

    tflog.Debug(ctx, "Creating InfluxDB client")
    client := influxdb2.NewClientWithOptions(url, token, opts)

    ready, err := client.Ready(ctx)  // Uses context from request!
    if err != nil {
        resp.Diagnostics.AddError("Unable to Connect to InfluxDB Server", err.Error())
        return
    }

    resp.ResourceData = client
}
```

**Key Improvements:**
- ✅ Context-aware configuration
- ✅ Structured diagnostics instead of raw errors
- ✅ Type-safe config model
- ✅ Proper logging with tflog
- ✅ No unsafe type assertions

---

## 2. Resource Create Operation

### Before (SDK v2)
```go
func resourceBucketCreate(d *schema.ResourceData, meta interface{}) error {
    influx := meta.(influxdb2.Client)  // ❌ Unsafe type assertion

    desc := d.Get("description").(string)  // ❌ Could panic
    orgid := d.Get("org_id").(string)      // ❌ Could panic

    newBucket := &domain.Bucket{
        Description: &desc,
        Name:        d.Get("name").(string),  // ❌ Could panic
        OrgID:       &orgid,
    }

    result, err := influx.BucketsAPI().CreateBucket(context.Background(), newBucket)  // ❌ Ignores context
    if err != nil {
        return fmt.Errorf("error creating bucket: %v", err)  // ❌ Should use %w
    }

    d.SetId(*result.Id)
    d.Set("name", result.Name)  // ❌ Ignores error
    return nil
}
```

### After (Plugin Framework)
```go
func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan BucketResourceModel

    // ✅ Type-safe config retrieval with diagnostics
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // ✅ Type-safe value access
    desc := plan.Description.ValueString()
    orgID := plan.OrgID.ValueString()

    newBucket := &domain.Bucket{
        Description: &desc,
        Name:        plan.Name.ValueString(),
        OrgID:       &orgID,
    }

    tflog.Debug(ctx, "Creating bucket", map[string]any{"name": plan.Name.ValueString()})

    // ✅ Uses context from request
    result, err := r.client.BucketsAPI().CreateBucket(ctx, newBucket)
    if err != nil {
        // ✅ Structured error with helpful message
        resp.Diagnostics.AddError(
            "Error Creating Bucket",
            "Could not create bucket, unexpected error: "+err.Error(),
        )
        return
    }

    plan.ID = types.StringValue(*result.Id)

    // ✅ Comprehensive state setting with error checking
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

**Key Improvements:**
- ✅ No panic-prone type assertions
- ✅ Context propagation for cancellation
- ✅ Structured error diagnostics
- ✅ Type-safe model structs
- ✅ Proper logging
- ✅ Better error messages

---

## 3. Type Assertions

### Before (SDK v2)
```go
// ❌ Will panic if type is wrong
influx := meta.(influxdb2.Client)
desc := d.Get("description").(string)
orgid := d.Get("org_id").(string)
everySeconds := int64(rr["every_seconds"].(int))

// ❌ Nested unsafe assertions
perm := permission.(map[string]interface{})
res := resource.(map[string]interface{})
id = res["id"].(string)
```

### After (Plugin Framework)
```go
// ✅ Type-safe client configuration
func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    client, ok := req.ProviderData.(influxdb2.Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected influxdb2.Client, got: %T", req.ProviderData),
        )
        return
    }
    r.client = client
}

// ✅ Type-safe model access
type BucketResourceModel struct {
    Description    types.String `tfsdk:"description"`
    OrgID          types.String `tfsdk:"org_id"`
    RetentionRules types.Set    `tfsdk:"retention_rules"`
}

// Access values safely
desc := model.Description.ValueString()
orgID := model.OrgID.ValueString()

// ✅ Type-safe nested structure conversion
var rules []RetentionRuleModel
diags := rulesSet.ElementsAs(ctx, &rules, false)
if diags.HasError() {
    return nil, fmt.Errorf("error converting retention rules set")
}
```

**Key Improvements:**
- ✅ Compile-time type safety
- ✅ Runtime validation with clear errors
- ✅ No possibility of panics
- ✅ Better IDE support

---

## 4. Error Handling

### Before (SDK v2)
```go
// ❌ Using %v doesn't wrap errors
return fmt.Errorf("error creating bucket: %v", err)

// ❌ Using %e is invalid
return fmt.Errorf("error creating authorization: %e", err)

// ❌ No error checking
d.Set("name", result.Name)
d.Set("description", result.Description)
```

### After (Plugin Framework)
```go
// ✅ Error wrapping with %w
if err := r.readBucket(ctx, &plan); err != nil {
    resp.Diagnostics.AddError(
        "Error Reading Bucket After Creation",
        "Could not read bucket after creation: "+err.Error(),
    )
    return
}

// ✅ Structured diagnostics
resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
if resp.Diagnostics.HasError() {
    return
}

// ✅ Helper functions use error wrapping
func (r *BucketResource) readBucket(ctx context.Context, model *BucketResourceModel) error {
    result, err := r.client.BucketsAPI().FindBucketByID(ctx, model.ID.ValueString())
    if err != nil {
        return fmt.Errorf("error finding bucket: %w", err)  // ✅ Uses %w
    }
    // ...
    return nil
}
```

**Key Improvements:**
- ✅ Error wrapping with %w
- ✅ Multiple errors collected in diagnostics
- ✅ User-friendly error titles and details
- ✅ Proper error propagation

---

## 5. Logging

### Before (SDK v2)
```go
// ❌ Standard log package, doesn't integrate with Terraform
log.Printf("[DEBUG] influxdb url %s", d.Get("url").(string))
log.Printf("[DEBUG] permissions %v", permissions)
log.Printf("Server is ready !")
```

### After (Plugin Framework)
```go
// ✅ Structured logging with tflog
tflog.Debug(ctx, "Creating InfluxDB client")

tflog.Debug(ctx, "Creating bucket", map[string]any{
    "name": plan.Name.ValueString(),
})

tflog.Info(ctx, "InfluxDB client configured successfully", map[string]any{
    "status": string(*ready.Status),
})

tflog.Trace(ctx, "Created bucket", map[string]any{
    "id": plan.ID.ValueString(),
})

// ✅ Mask sensitive data
ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "influxdb_token")
```

**Key Improvements:**
- ✅ Integrates with Terraform's logging
- ✅ Structured log fields
- ✅ Proper log levels (Trace, Debug, Info, Warn, Error)
- ✅ Can mask sensitive data
- ✅ Context-aware

---

## 6. Context Usage

### Before (SDK v2)
```go
// ❌ Always uses Background context, ignoring timeouts/cancellation
result, err := influx.BucketsAPI().CreateBucket(context.Background(), newBucket)
ready, err := influx.Ready(context.Background())
_, err = influx.AuthorizationsAPI().CreateAuthorization(context.Background(), &auth)
```

### After (Plugin Framework)
```go
// ✅ Uses context from request, enabling proper cancellation
func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    result, err := r.client.BucketsAPI().CreateBucket(ctx, newBucket)
    // ...
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    result, err := r.client.BucketsAPI().FindBucketByID(ctx, state.ID.ValueString())
    // ...
}

func (d *ReadyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    ready, err := d.client.Ready(ctx)
    // ...
}
```

**Key Improvements:**
- ✅ Respects context cancellation
- ✅ Timeout support works properly
- ✅ Can be canceled by user (Ctrl+C)
- ✅ Better resource cleanup

---

## 7. Schema Definition

### Before (SDK v2)
```go
Schema: map[string]*schema.Schema{
    "name": {
        Type:     schema.TypeString,
        Required: true,
    },
    "retention_rules": {
        Type:     schema.TypeSet,
        Required: true,
        Elem: &schema.Resource{
            Schema: map[string]*schema.Schema{
                "every_seconds": {
                    Type:     schema.TypeInt,
                    Required: true,
                },
            },
        },
    },
}
```

### After (Plugin Framework)
```go
resp.Schema = schema.Schema{
    Description: "Manages an InfluxDB v2 bucket.",
    Attributes: map[string]schema.Attribute{
        "name": schema.StringAttribute{
            Description: "The name of the bucket.",
            Required:    true,
        },
        "retention_rules": schema.SetNestedAttribute{
            Description: "Retention rules for the bucket.",
            Required:    true,
            NestedObject: schema.NestedAttributeObject{
                Attributes: map[string]schema.Attribute{
                    "every_seconds": schema.Int64Attribute{
                        Description: "Duration in seconds for how long data will be kept.",
                        Required:    true,
                        PlanModifiers: []planmodifier.Int64{
                            int64planmodifier.UseStateForUnknown(),
                        },
                    },
                },
            },
        },
    },
}
```

**Key Improvements:**
- ✅ Strongly-typed attributes
- ✅ Built-in descriptions for documentation
- ✅ Plan modifiers for behavior control
- ✅ Better validation support
- ✅ Clearer structure

---

## 8. Import Support

### Before (SDK v2)
```go
// ❌ No import support - users couldn't import existing resources
func ResourceBucket() *schema.Resource {
    return &schema.Resource{
        Create: resourceBucketCreate,
        Read:   resourceBucketRead,
        Update: resourceBucketUpdate,
        Delete: resourceBucketDelete,
        // No Importer field
    }
}
```

### After (Plugin Framework)
```go
// ✅ Full import support
var _ resource.ResourceWithImportState = &BucketResource{}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Usage:**
```bash
terraform import influxdb-v2_bucket.example <bucket-id>
```

---

## Summary of Improvements

| Aspect | Before | After |
|--------|--------|-------|
| Type Safety | ❌ Unsafe assertions | ✅ Compile-time types |
| Error Handling | ❌ Basic fmt.Errorf | ✅ Structured diagnostics |
| Logging | ❌ Standard log | ✅ tflog with structure |
| Context | ❌ Background only | ✅ Full propagation |
| Import | ❌ Not supported | ✅ Full support |
| Validation | ❌ Manual | ✅ Built-in validators |
| Documentation | ❌ Separate | ✅ In schema |
| Plan Modifiers | ❌ Not available | ✅ Full support |
