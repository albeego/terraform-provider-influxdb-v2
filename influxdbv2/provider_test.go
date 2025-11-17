package influxdbv2

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"influxdb-v2": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Ensure required environment variables are set for acceptance tests
	if v := os.Getenv("INFLUXDB_V2_URL"); v == "" {
		t.Fatal("INFLUXDB_V2_URL must be set for acceptance tests")
	}
	if v := os.Getenv("INFLUXDB_V2_TOKEN"); v == "" {
		t.Fatal("INFLUXDB_V2_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("INFLUXDB_V2_ORG_ID"); v == "" {
		t.Fatal("INFLUXDB_V2_ORG_ID must be set for acceptance tests")
	}
}
