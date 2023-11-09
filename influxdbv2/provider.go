package influxdbv2

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/influxdata/influxdb-client-go/v2"
	"log"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"influxdb-v2_ready": DataReady(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"influxdb-v2_bucket":        ResourceBucket(),
			"influxdb-v2_authorization": ResourceAuthorization(),
		},
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("INFLUXDB_V2_URL", "http://localhost:8086"),
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("INFLUXDB_V2_TOKEN", ""),
			},
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	opts := influxdb2.DefaultOptions().SetLogLevel(2)
	influx := influxdb2.NewClientWithOptions(d.Get("url").(string), d.Get("token").(string), opts)

	log.Printf("[DEBUG] influxdb url %s", d.Get("url").(string))
	_, err := influx.Ready(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error pinging server: %s", err)
	}

	return influx, nil
}
