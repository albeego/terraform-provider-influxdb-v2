package influxdbv2

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBucketResource(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccBucketResourceConfig("test-bucket", "Test bucket description", orgID, 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "name", "test-bucket"),
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "description", "Test bucket description"),
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "org_id", orgID),
					resource.TestCheckResourceAttrSet("influxdb-v2_bucket.test", "id"),
					resource.TestCheckResourceAttrSet("influxdb-v2_bucket.test", "created_at"),
					resource.TestCheckResourceAttrSet("influxdb-v2_bucket.test", "updated_at"),
					resource.TestCheckResourceAttrSet("influxdb-v2_bucket.test", "type"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "influxdb-v2_bucket.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccBucketResourceConfig("test-bucket-updated", "Updated description", orgID, 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "name", "test-bucket-updated"),
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccBucketResource_WithRetentionPolicy(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketResourceConfigWithRP("test-bucket-rp", "Bucket with RP", orgID, 3600, "autogen"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "name", "test-bucket-rp"),
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "rp", "autogen"),
				),
			},
		},
	})
}

func TestAccBucketResource_MultipleRetentionRules(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketResourceConfigMultipleRules("test-bucket-multi", "Multiple rules", orgID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_bucket.test", "name", "test-bucket-multi"),
					resource.TestCheckTypeSetElemNestedAttrs("influxdb-v2_bucket.test", "retention_rules.*", map[string]string{
						"every_seconds": "3600",
						"type":          "expire",
					}),
				),
			},
		},
	})
}

func testAccBucketResourceConfig(name, description, orgID string, everySeconds int) string {
	return fmt.Sprintf(`
resource "influxdb-v2_bucket" "test" {
  name        = %[1]q
  description = %[2]q
  org_id      = %[3]q

  retention_rules {
    every_seconds = %[4]d
    type          = "expire"
  }
}
`, name, description, orgID, everySeconds)
}

func testAccBucketResourceConfigWithRP(name, description, orgID string, everySeconds int, rp string) string {
	return fmt.Sprintf(`
resource "influxdb-v2_bucket" "test" {
  name        = %[1]q
  description = %[2]q
  org_id      = %[3]q
  rp          = %[5]q

  retention_rules {
    every_seconds = %[4]d
    type          = "expire"
  }
}
`, name, description, orgID, everySeconds, rp)
}

func testAccBucketResourceConfigMultipleRules(name, description, orgID string) string {
	return fmt.Sprintf(`
resource "influxdb-v2_bucket" "test" {
  name        = %[1]q
  description = %[2]q
  org_id      = %[3]q

  retention_rules {
    every_seconds = 3600
    type          = "expire"
  }
}
`, name, description, orgID)
}

// Helper function to check if bucket exists
func testAccCheckBucketExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Bucket ID is set")
		}

		return nil
	}
}
