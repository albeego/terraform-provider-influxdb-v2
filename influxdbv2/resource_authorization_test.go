package influxdbv2

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAuthorizationResource(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")
	bucketID := os.Getenv("INFLUXDB_V2_BUCKET_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAuthorizationResourceConfig(orgID, bucketID, "active", "Test authorization"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "org_id", orgID),
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "status", "active"),
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "description", "Test authorization"),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "id"),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "token"),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "user_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "influxdb-v2_authorization.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Token is not returned on subsequent reads, and permissions aren't fully readable via API
				ImportStateVerifyIgnore: []string{"token", "permissions", "description", "org_id"},
			},
			// Update status to inactive
			{
				Config: testAccAuthorizationResourceConfig(orgID, bucketID, "inactive", "Test authorization"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "status", "inactive"),
				),
			},
			// Update status back to active
			{
				Config: testAccAuthorizationResourceConfig(orgID, bucketID, "active", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "status", "active"),
				),
			},
		},
	})
}

func TestAccAuthorizationResource_ReadPermission(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")
	bucketID := os.Getenv("INFLUXDB_V2_BUCKET_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthorizationResourceConfigReadOnly(orgID, bucketID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "org_id", orgID),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "id"),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "token"),
				),
			},
		},
	})
}

func TestAccAuthorizationResource_WritePermission(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")
	bucketID := os.Getenv("INFLUXDB_V2_BUCKET_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthorizationResourceConfigWriteOnly(orgID, bucketID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "org_id", orgID),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "id"),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "token"),
				),
			},
		},
	})
}

func TestAccAuthorizationResource_MultiplePermissions(t *testing.T) {
	orgID := os.Getenv("INFLUXDB_V2_ORG_ID")
	bucketID := os.Getenv("INFLUXDB_V2_BUCKET_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthorizationResourceConfigMultiplePermissions(orgID, bucketID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb-v2_authorization.test", "org_id", orgID),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "id"),
					resource.TestCheckResourceAttrSet("influxdb-v2_authorization.test", "token"),
				),
			},
		},
	})
}

func testAccAuthorizationResourceConfig(orgID, bucketID, status, description string) string {
	return fmt.Sprintf(`
resource "influxdb-v2_authorization" "test" {
  org_id      = %[1]q
  status      = %[3]q
  description = %[4]q

  permissions {
    action = "read"
    resource {
      id     = %[2]q
      org_id = %[1]q
      type   = "buckets"
    }
  }

  permissions {
    action = "write"
    resource {
      id     = %[2]q
      org_id = %[1]q
      type   = "buckets"
    }
  }
}
`, orgID, bucketID, status, description)
}

func testAccAuthorizationResourceConfigReadOnly(orgID, bucketID string) string {
	return fmt.Sprintf(`
resource "influxdb-v2_authorization" "test" {
  org_id      = %[1]q
  status      = "active"
  description = "Read-only authorization"

  permissions {
    action = "read"
    resource {
      id     = %[2]q
      org_id = %[1]q
      type   = "buckets"
    }
  }
}
`, orgID, bucketID)
}

func testAccAuthorizationResourceConfigWriteOnly(orgID, bucketID string) string {
	return fmt.Sprintf(`
resource "influxdb-v2_authorization" "test" {
  org_id      = %[1]q
  status      = "active"
  description = "Write-only authorization"

  permissions {
    action = "write"
    resource {
      id     = %[2]q
      org_id = %[1]q
      type   = "buckets"
    }
  }
}
`, orgID, bucketID)
}

func testAccAuthorizationResourceConfigMultiplePermissions(orgID, bucketID string) string {
	return fmt.Sprintf(`
resource "influxdb-v2_authorization" "test" {
  org_id      = %[1]q
  status      = "active"
  description = "Authorization with multiple permissions"

  permissions {
    action = "read"
    resource {
      id     = %[2]q
      org_id = %[1]q
      type   = "buckets"
    }
  }

  permissions {
    action = "write"
    resource {
      id     = %[2]q
      org_id = %[1]q
      type   = "buckets"
    }
  }

  permissions {
    action = "read"
    resource {
      id     = %[1]q
      org_id = %[1]q
      type   = "orgs"
    }
  }
}
`, orgID, bucketID)
}
