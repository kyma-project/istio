# Variables for integration tests run on a Gardener GCP ipv4 shoot.
# Auto-loaded by provision.sh / integration-test.sh when
# GARDENER_CONFIGURATION_PRESET=gcp-ipv4.

MACHINE_TYPE="n2-standard-4"
DISK_SIZE=50
DISK_TYPE="pd-standard"
SCALER_MAX=3
SCALER_MIN=1
GARDENER_PROVIDER="gcp"
GARDENER_IP_STACK="ipv4"
GARDENER_REGION="europe-west3"
GARDENER_PROVIDER_SECRET_NAME="goat-gcp-secret"
GARDENER_CLUSTER_VERSION="1.33.5"
