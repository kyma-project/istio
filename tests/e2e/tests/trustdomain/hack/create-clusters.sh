#!/usr/bin/env bash
set -euo pipefail

cd ..

K3D_PREFIX="k3d"
CLUSTER1_NAME="kyma1"
CLUSTER2_NAME="kyma2"
NETWORK_NAME="kyma"
CTX_CLUSTER1="${K3D_PREFIX}-${CLUSTER1_NAME}"
CTX_CLUSTER2="${K3D_PREFIX}-${CLUSTER2_NAME}"
LOAD_BALANCER_NAME_CLUSTER2=$(k3d cluster list "${CLUSTER2_NAME}" -o json | jq -r ".[0].serverLoadBalancer.name")
WORKLOAD_DOMAIN="${LOAD_BALANCER_NAME_CLUSTER2}.${NETWORK_NAME}"

echo "Cleanup old clusters"
k3d cluster delete "${CLUSTER1_NAME}" || true
k3d cluster delete "${CLUSTER2_NAME}" || true
docker network rm "${NETWORK_NAME}" || true

echo "Create network"
docker network create -d bridge "${NETWORK_NAME}"

echo "Create Cluster 1 - client side"
k3d cluster create "${CLUSTER1_NAME}" --network "${NETWORK_NAME}" --port 1080:80@loadbalancer --port 1443:443@loadbalancer  --image rancher/k3s:v1.33.3-k3s1 --k3s-arg "--disable=traefik@server:*" --k3s-arg "--cluster-cidr=10.10.0.0/16@server:*" --k3s-arg "--service-cidr=10.11.0.0/16@server:*"

echo "Create Cluster 2 - server side"
k3d cluster create "${CLUSTER2_NAME}" --network "${NETWORK_NAME}" --port 2080:80@loadbalancer --port 2443:443@loadbalancer  --image rancher/k3s:v1.33.3-k3s1 --k3s-arg "--disable=traefik@server:*" --k3s-arg "--cluster-cidr=10.20.0.0/16@server:*" --k3s-arg "--service-cidr=10.21.0.0/16@server:*"

export KUBECONFIG1="$(k3d kubeconfig write "${CLUSTER1_NAME}")"
export KUBECONFIG2="$(k3d kubeconfig write "${CLUSTER2_NAME}")"