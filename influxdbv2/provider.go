package influxdbv2

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &influxdbProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &influxdbProvider{
			version: version,
		}
	}
}

// influxdbProvider is the provider implementation.
type influxdbProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance testing.
	version string
}

// influxdbProviderModel describes the provider data model.
type influxdbProviderModel struct {
	URL   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

// Metadata returns the provider type name.
func (p *influxdbProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "influxdb-v2"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *influxdbProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing InfluxDB v2 resources.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "InfluxDB server URL. Can also be set via INFLUXDB_V2_URL environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "InfluxDB authentication token. Can also be set via INFLUXDB_V2_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares a InfluxDB API client for data sources and resources.
func (p *influxdbProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring InfluxDB v2 client")

	// Retrieve provider data from configuration
	var config influxdbProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	url := os.Getenv("INFLUXDB_V2_URL")
	if !config.URL.IsNull() {
		url = config.URL.ValueString()
	}
	if url == "" {
		url = "http://localhost:8086"
	}

	token := os.Getenv("INFLUXDB_V2_TOKEN")
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if url == "" {
		resp.Diagnostics.AddError(
			"Missing InfluxDB URL Configuration",
			"While configuring the provider, the InfluxDB URL was not found in "+
				"the INFLUXDB_V2_URL environment variable or provider "+
				"configuration block url attribute.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing InfluxDB Token Configuration",
			"While configuring the provider, the InfluxDB token was not found in "+
				"the INFLUXDB_V2_TOKEN environment variable or provider "+
				"configuration block token attribute.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "influxdb_url", url)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "influxdb_token")

	tflog.Debug(ctx, "Creating InfluxDB client")

	// Create InfluxDB client
	opts := influxdb2.DefaultOptions().SetLogLevel(2)
	client := influxdb2.NewClientWithOptions(url, token, opts)

	// Verify connection to InfluxDB
	ready, err := client.Ready(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Connect to InfluxDB Server",
			"An unexpected error occurred when connecting to the InfluxDB server. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"InfluxDB Client Error: "+err.Error(),
		)
		return
	}

	if ready == nil || ready.Status == nil {
		resp.Diagnostics.AddError(
			"InfluxDB Server Not Ready",
			"The InfluxDB server is not ready to accept connections.",
		)
		return
	}

	tflog.Info(ctx, "InfluxDB client configured successfully", map[string]any{"status": string(*ready.Status)})

	// Make the InfluxDB client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *influxdbProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewReadyDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *influxdbProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBucketResource,
		NewAuthorizationResource,
	}
}
