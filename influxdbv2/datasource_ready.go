package influxdbv2

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ReadyDataSource{}

func NewReadyDataSource() datasource.DataSource {
	return &ReadyDataSource{}
}

// ReadyDataSource defines the data source implementation.
type ReadyDataSource struct {
	client influxdb2.Client
}

// ReadyDataSourceModel describes the data source data model.
type ReadyDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	URL     types.String `tfsdk:"url"`
	Ready   types.Bool   `tfsdk:"ready"`
	Status  types.String `tfsdk:"status"`
	Started types.String `tfsdk:"started"`
}

func (d *ReadyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ready"
}

func (d *ReadyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source to check if the InfluxDB server is ready to accept connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Data source identifier (server URL).",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The InfluxDB server URL.",
				Computed:    true,
			},
			"ready": schema.BoolAttribute{
				Description: "Whether the server is ready.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The server status.",
				Computed:    true,
			},
			"started": schema.StringAttribute{
				Description: "Timestamp when the server started.",
				Computed:    true,
			},
		},
	}
}

func (d *ReadyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(influxdb2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected influxdb2.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ReadyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ReadyDataSourceModel

	tflog.Debug(ctx, "Checking if InfluxDB server is ready")

	// Check if server is ready
	ready, err := d.client.Ready(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Checking Server Status",
			"Could not check if server is ready: "+err.Error(),
		)
		return
	}

	// Get server URL
	serverURL := d.client.ServerURL()

	// Populate model
	state.ID = types.StringValue(serverURL)
	state.URL = types.StringValue(serverURL)
	state.Ready = types.BoolValue(true) // If we got here, server is ready

	if ready.Status != nil {
		state.Status = types.StringValue(string(*ready.Status))
	} else {
		state.Status = types.StringValue("unknown")
	}

	if ready.Started != nil {
		if ready.Started.Before(time.Now()) {
			tflog.Info(ctx, "Server is ready", map[string]any{
				"url":     serverURL,
				"started": ready.Started.String(),
			})
		}
		state.Started = types.StringValue(ready.Started.String())
	} else {
		state.Started = types.StringValue("")
	}

	tflog.Trace(ctx, "InfluxDB server ready check completed", map[string]any{
		"url":   serverURL,
		"ready": true,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
