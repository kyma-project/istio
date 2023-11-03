#!/usr/bin/env bash

set -e

export MACHINE_TYPE="m5.xlarge"
export DISK_SIZE=50
export DISK_TYPE="gp2"
export SCALER_MAX=3
export SCALER_MIN=1
export GARDENER_PROVIDER="aws"
export GARDENER_REGION="eu-north-1"
export GARDENER_ZONES="eu-north-1b,eu-north-1c,eu-north-1a"

./tests/integration/scripts/integration-test.sh aws-integration-test