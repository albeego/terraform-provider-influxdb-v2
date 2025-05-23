terraform {
  required_providers {
    influxdb-v2 = {
      source = "albeego/influxdb-v2"
      version = "0.1.0"
    }
  }
}

provider "influxdb-v2" {
    // provider is configured with env vars
}

variable "influx_org_id" {
  description = "Influxdb organization ID defined at the onboarding stage"
}

data "influxdb-v2_ready" "status" {}

output "influxdb-v2_is_ready" {
    value = data.influxdb-v2_ready.status.output["url"]
}

resource "influxdb-v2_bucket" "temp" {
    description = "Temperature sensors data"
    name = "temp"
    org_id = var.influx_org_id
    retention_rules {
        every_seconds = 3600*24*30
    }
}

resource "influxdb-v2_authorization" "api" {
    org_id = var.influx_org_id
    description = "api token"
    status = "active"
    permissions {
        action = "read"
        resource {
            id = influxdb-v2_bucket.temp.id
            org_id = var.influx_org_id
            type = "buckets"
        }
    }
    permissions {
        action = "write"
        resource {
            id = influxdb-v2_bucket.temp.id
            org_id = var.influx_org_id
            type = "buckets"
        }
    }
}

output "api_token" {
    value = influxdb-v2_authorization.api.token
}
