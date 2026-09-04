# Variables for the Kubernetes version compatibility check on a Gardener AWS
# ipv4 shoot. Auto-loaded by provision.sh / integration-test.sh
# when GARDENER_CONFIGURATION=aws-ipv4-compatibility. Same shoot as aws-ipv4, but pinned to
# a newer Kubernetes version to catch upcoming-version breakages early.

MACHINE_TYPE="m5.xlarge"
DISK_SIZE=50
DISK_TYPE="gp2"
SCALER_MAX=3
SCALER_MIN=3 # we need at least 3 machines, because there are 3 availability zones
GARDENER_PROVIDER="aws"
GARDENER_IP_STACK="ipv4"
GARDENER_REGION="eu-west-1"
GARDENER_PROVIDER_SECRET_NAME="goat-aws-secret"
GARDENER_CLUSTER_VERSION="1.36.3"
