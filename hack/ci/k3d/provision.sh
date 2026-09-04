#!/usr/bin/env bash

# Description: This script downloads k3d CLI and provisions a k3d cluster
# Environment variables (optional):
#   KUBERNETES_VERSION  - Kubernetes version (default: 1.33.5)
#   K3D_VERSION         - k3d CLI version (default: v5.7.5)
#   CALICO_VERSION      - Calico version for --calico mode (default: v3.29.0)
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
AGENTS="${AGENTS:-0}"
SERVERS_MEMORY="${SERVERS_MEMORY:-16}"
SERVERS="${SERVERS:-1}"

# Parse flags
USE_CALICO=false
USE_KWOK=false
KWOK_NODES="${10:-0}"
while [[ $# -gt 0 ]]; do
    case "${1}" in
        --calico)
            USE_CALICO=true
            ;;
        --kwok)
            USE_KWOK=true
            ;;
        --kwok-nodes)
            KWOK_NODES="${2}"
            shift
            ;;
        *)
            echo "Unknown argument: ${1}"
            exit 1
            ;;
    esac
    shift
done

# Construct the k3s image tag
K3S_IMAGE="rancher/k3s:v${KUBERNETES_VERSION}-k3s1"

echo "Configuration:"
echo "  Kubernetes version: ${KUBERNETES_VERSION}"
echo "  K3s image: ${K3S_IMAGE}"
echo "  Use Calico: ${USE_CALICO}"
echo "  k3d version: ${K3D_VERSION}"
echo "  Agents: ${AGENTS}"
echo "  Servers: ${SERVERS}"
echo "  Servers memory: ${SERVERS_MEMORY}g"

# Function to install k3d
install_k3d() {
    if command -v k3d &> /dev/null; then
        echo "k3d is already installed: $(k3d version | head -1)"
        return
    fi

    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

    echo "k3d installed successfully: $(k3d version | head -1)"
}

# Function to provision cluster with Calico
provision_calico_cluster() {
    if [ "${AGENTS}" -gt 0 ]; then
        echo "Error: script does not support calico setup with AGENTS > 0. Please use AGENTS=0."
        exit 1
    fi

    echo "Provisioning k3d cluster with Calico CNI..."

    k3d cluster create \
        --agents "${AGENTS}" \
        --servers "${SERVERS}" \
        --servers-memory "${SERVERS_MEMORY}g" \
        --port 80:80@loadbalancer \
        --port 443:443@loadbalancer \
        --k3s-arg "--flannel-backend=none@all" \
        --k3s-arg "--disable=traefik@server:*" \
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

    k3d cluster create \
        --agents "${AGENTS}" \
        --servers-memory "${SERVERS_MEMORY}g" \
        --port 80:80@loadbalancer \
        --port 443:443@loadbalancer \
        --k3s-arg '--disable=traefik@server:*' \
        --image "${K3S_IMAGE}"
}

setup_kwok() {
# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
KWOK_RELEASE="v0.7.0"
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_RELEASE}/kwok.yaml"
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_RELEASE}/stage-fast.yaml"
kubectl apply -f "hack/manifests/chaos/job-pod-running.yaml"

if [[ "${KWOK_NODES}" -gt 0 ]]; then
    echo "Creating ${KWOK_NODES} fake Nodes..."
    for i in $(seq 1 "${KWOK_NODES}"); do
kubectl apply -f - <<EOF
  apiVersion: v1
  kind: Node
  metadata:
    annotations:
      node.alpha.kubernetes.io/ttl: "0"
      kwok.x-k8s.io/node: fake
    labels:
      beta.kubernetes.io/arch: amd64
      beta.kubernetes.io/os: linux
      kubernetes.io/arch: amd64
      kubernetes.io/hostname: kwok-node-$i
      kubernetes.io/os: linux
      kubernetes.io/role: agent
      node-role.kubernetes.io/agent: ""
      type: kwok
    name: kwok-node-$i
  spec:
    taints: # Avoid scheduling actual running pods to fake Node
    - effect: NoSchedule
      key: kwok.x-k8s.io/node
      value: fake
  status:
    allocatable:
      cpu: 32
      memory: 250Gi
      pods: 110
    capacity:
      cpu: 32
      memory: 256Gi
      pods: 110
    nodeInfo:
      architecture: amd64
      bootID: ""
      containerRuntimeVersion: ""
      kernelVersion: ""
      kubeProxyVersion: "fake"
      kubeletVersion: v1.35.0
      machineID: ""
      operatingSystem: linux
      osImage: "Debian GNU/Linux 12 (trixie)"
      systemUUID: ""
    phase: Running
EOF
  done
  fi
}

# Main execution
main() {
    install_k3d

    if [ "${USE_CALICO}" = true ]; then
        provision_calico_cluster
    else
        provision_regular_cluster
    fi

    if [ "${USE_KWOK}" = true ]; then
        setup_kwok
    fi
}

main
