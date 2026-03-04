#!/usr/bin/env bash

# Description: This script downloads k3d CLI and provisions a k3d cluster
# Environment variables (optional):
#   KUBERNETES_VERSION  - Kubernetes version (default: 1.33.5)
#   K3D_VERSION         - k3d CLI version (default: v5.7.5)
#   CALICO_VERSION      - Calico version for --calico mode (default: v3.29.0)
#   CLUSTER_NAME        - Cluster name (default: kyma)
#   AGENTS              - Number of k3d agents (default: 0)
#   SERVERS_MEMORY      - Memory for server nodes in GB (default: 16)

set -eo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Configuration - override via environment variables
KUBERNETES_VERSION="${KUBERNETES_VERSION:-1.34.3}"
K3D_VERSION="${K3D_VERSION:-v5.8.3}"
CALICO_VERSION="${CALICO_VERSION:-v3.31.3}"
CLUSTER_NAME="${CLUSTER_NAME:-kyma}"
AGENTS="${AGENTS:-0}"
SERVERS_MEMORY="${SERVERS_MEMORY:-16}"

# Parse --calico flag
USE_CALICO=false
if [[ "${1:-}" == "--calico" ]]; then
    USE_CALICO=true
fi

# Construct the k3s image tag
K3S_IMAGE="rancher/k3s:v${KUBERNETES_VERSION}-k3s1"

echo "Configuration:"
echo "  Cluster name: ${CLUSTER_NAME}"
echo "  Kubernetes version: ${KUBERNETES_VERSION}"
echo "  K3s image: ${K3S_IMAGE}"
echo "  Use Calico: ${USE_CALICO}"
echo "  k3d version: ${K3D_VERSION}"
echo "  Agents: ${AGENTS}"
echo "  Servers memory: ${SERVERS_MEMORY}g"

# Function to install k3d
install_k3d() {
    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

    echo "k3d installed successfully: $(k3d version | head -1)"
}

# Function to provision cluster with Calico
provision_calico_cluster() {
    echo "Provisioning k3d cluster with Calico CNI..."

    k3d cluster create "${CLUSTER_NAME}" \
        --agents "${AGENTS}" \
        --servers-memory "${SERVERS_MEMORY}g" \
        --port 80:80@loadbalancer \
        --port 443:443@loadbalancer \
        --k3s-arg "--flannel-backend=none@all" \
        --k3s-arg "--disable=traefik@server:0" \
        --k3s-arg '--tls-san=host.docker.internal@server:*' \
        --image "${K3S_IMAGE}"

    echo "Installing Calico ${CALICO_VERSION}..."
    kubectl create -f "https://raw.githubusercontent.com/projectcalico/calico/${CALICO_VERSION}/manifests/operator-crds.yaml"
    kubectl create -f "https://raw.githubusercontent.com/projectcalico/calico/${CALICO_VERSION}/manifests/tigera-operator.yaml"
    kubectl create -f "https://raw.githubusercontent.com/projectcalico/calico/${CALICO_VERSION}/manifests/custom-resources.yaml"

    kubectl rollout status -n kube-system deployment coredns
    kubectl patch installation default --type=merge -p '{"spec":{"cni":{"binDir":"/var/lib/rancher/k3s/data/cni", "confDir":"/var/lib/rancher/k3s/agent/etc/cni/net.d"}}}'

}

# Function to provision regular cluster (without traefik)
provision_regular_cluster() {
    echo "Provisioning k3d cluster (regular, without traefik)..."

    k3d cluster create "${CLUSTER_NAME}" \
        --agents "${AGENTS}" \
        --servers-memory "${SERVERS_MEMORY}g" \
        --port 80:80@loadbalancer \
        --port 443:443@loadbalancer \
        --k3s-arg '--disable=traefik@server:*' \
        --image "${K3S_IMAGE}"
}

# Main execution
main() {
    install_k3d

    # Check if cluster already exists
    if k3d cluster list | grep -q "^${CLUSTER_NAME} "; then
        echo "Cluster '${CLUSTER_NAME}' already exists."
        echo "Aborting."
        exit 0
    fi

    if [ "${USE_CALICO}" = true ]; then
        provision_calico_cluster
    else
        provision_regular_cluster
    fi

    # Set kubeconfig
    echo "Setting up kubeconfig..."
    k3d kubeconfig merge "${CLUSTER_NAME}" -d --kubeconfig-switch-context

    # Wait for nodes to be ready
    echo "Waiting for nodes to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s

    echo ""
    echo "=========================================="
    echo "k3d cluster '${CLUSTER_NAME}' provisioned successfully!"
    echo "Kubernetes version: ${KUBERNETES_VERSION}"
    if [ "${USE_CALICO}" = true ]; then
        echo "CNI: Calico ${CALICO_VERSION}"
    else
        echo "CNI: Flannel (k3s default)"
    fi
    echo "Traefik disabled"
    echo "=========================================="
}

main
