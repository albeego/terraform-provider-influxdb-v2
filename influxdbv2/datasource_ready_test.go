package influxdbv2

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReadyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccReadyDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.influxdb-v2_ready.test", "id"),
					resource.TestCheckResourceAttrSet("data.influxdb-v2_ready.test", "url"),
					resource.TestCheckResourceAttr("data.influxdb-v2_ready.test", "ready", "true"),
					resource.TestCheckResourceAttrSet("data.influxdb-v2_ready.test", "status"),
					resource.TestCheckResourceAttrSet("data.influxdb-v2_ready.test", "started"),
				),
			},
		},
	})
}

func TestAccReadyDataSource_URL(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReadyDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.influxdb-v2_ready.test",
						"url",
						regexp.MustCompile(`^http://`),
					),
				),
			},
		},
	})
}

func TestAccReadyDataSource_Status(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReadyDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.influxdb-v2_ready.test",
						"status",
						regexp.MustCompile(`ready`),
					),
				),
			},
		},
	})
}

const testAccReadyDataSourceConfig = `
data "influxdb-v2_ready" "test" {}
`
