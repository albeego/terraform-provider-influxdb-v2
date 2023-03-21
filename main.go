package main

import (
	"github.com/albeego/terraform-provider-influxdb-v2/influxdbv2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: influxdbv2.Provider})
}
