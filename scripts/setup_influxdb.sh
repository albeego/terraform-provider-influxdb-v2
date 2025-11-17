#!/usr/bin/env bash

set -e

echo "======================================"
echo "InfluxDB v2 Test Setup"
echo "======================================"

# Check if container already exists
if docker ps -a --format '{{.Names}}' | grep -q '^tf_acc_tests_influxdb$'; then
    echo "1) Removing existing InfluxDB container..."
    docker stop tf_acc_tests_influxdb 2>/dev/null || true
    docker rm tf_acc_tests_influxdb 2>/dev/null || true
fi

echo "2) Launching InfluxDB container..."
docker run -d --name tf_acc_tests_influxdb -p 8086:8086 influxdb:2.7

echo "3) Waiting for InfluxDB to be ready..."
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -sS 'http://localhost:8086/ready' | grep -q ready; then
        echo "   ✓ InfluxDB is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo "   Waiting... (attempt $attempt/$max_attempts)"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "   ✗ InfluxDB failed to become ready"
    exit 1
fi

echo "4) Running onboarding setup..."
onboard=$(curl -fsSL -X POST \
    --data '{"username":"admin", "password":"password123", "org":"testorg", "bucket":"testbucket", "retentionPeriodHrs":0}' \
    "http://localhost:8086/api/v2/setup")

echo "5) Extracting configuration..."
token=$(echo $onboard | jq -Mcr '.auth.token')
bucketid=$(echo $onboard | jq -Mcr '.bucket.id')
orgid=$(echo $onboard | jq -Mcr '.bucket.orgID')

echo "======================================"
echo "Setup Complete!"
echo "======================================"
echo "URL:       http://localhost:8086"
echo "Org ID:    $orgid"
echo "Bucket ID: $bucketid"
echo "Token:     ${token:0:20}..."
echo "======================================"

echo ""
echo "Export these environment variables:"
echo "export INFLUXDB_V2_URL=\"http://localhost:8086\""
echo "export INFLUXDB_V2_TOKEN=\"$token\""
echo "export INFLUXDB_V2_BUCKET_ID=\"$bucketid\""
echo "export INFLUXDB_V2_ORG_ID=\"$orgid\""
echo ""

# Export for current shell
export INFLUXDB_V2_URL="http://localhost:8086"
export INFLUXDB_V2_TOKEN="$token"
export INFLUXDB_V2_BUCKET_ID="$bucketid"
export INFLUXDB_V2_ORG_ID="$orgid"

