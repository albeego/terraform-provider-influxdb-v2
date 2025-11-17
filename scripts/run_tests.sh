#!/usr/bin/env bash

set -e

echo "======================================"
echo "Running InfluxDB v2 Provider Tests"
echo "======================================"

# Source the setup script to get environment variables
echo "Setting up InfluxDB test environment..."
source ./scripts/setup_influxdb.sh

echo ""
echo "======================================"
echo "Running Acceptance Tests"
echo "======================================"

# Run acceptance tests
TF_ACC=1 go test -v -timeout 10m ./influxdbv2/...

echo ""
echo "======================================"
echo "Tests Complete!"
echo "======================================"
echo ""
echo "To clean up, run:"
echo "  docker stop tf_acc_tests_influxdb"
echo "  docker rm tf_acc_tests_influxdb"
