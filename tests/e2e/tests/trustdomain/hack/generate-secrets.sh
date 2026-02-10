#!/usr/bin/env bash
set -euo pipefail

cd ..

# This script generates the secrets for the trust domain tests.
echo "Generate root CA"
export ROOT_CA_CN="Root CA"
export ROOT_CA_ORG="Org"
export ROOT_CA_KEY_FILE="${ROOT_CA_CN}.key"
export ROOT_CA_CRT_FILE="${ROOT_CA_CN}.crt"
openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj "/O=${ROOT_CA_ORG}/CN=${ROOT_CA_CN}" -keyout "${ROOT_CA_KEY_FILE}" -out "${ROOT_CA_CRT_FILE}" \
-addext "basicConstraints = critical, CA:true"

echo "Generate intermediate CA for cluster1"
export CLUSTER1_INTERMEDIATE_CA_CN="Cluster 1 CA"
export CLUSTER1_INTERMEDIATE_CA_ORG="Org"
export CLUSTER1_INTERMEDIATE_CA_CRT_FILE="${CLUSTER1_INTERMEDIATE_CA_CN}.crt"
export CLUSTER1_INTERMEDIATE_CA_CSR_FILE="${CLUSTER1_INTERMEDIATE_CA_CN}.csr"
export CLUSTER1_INTERMEDIATE_CA_KEY_FILE="${CLUSTER1_INTERMEDIATE_CA_CN}.key"
openssl req -out "${CLUSTER1_INTERMEDIATE_CA_CSR_FILE}" -newkey rsa:2048 -nodes -keyout "${CLUSTER1_INTERMEDIATE_CA_KEY_FILE}" -subj "/CN=${CLUSTER1_INTERMEDIATE_CA_CN}/O=${CLUSTER1_INTERMEDIATE_CA_ORG}"

echo "Sign intermediate CA for cluster1 with root CA"
openssl x509 -req -days 365 -CA "${ROOT_CA_CRT_FILE}" -CAkey "${ROOT_CA_KEY_FILE}" -set_serial 1 -in "${CLUSTER1_INTERMEDIATE_CA_CSR_FILE}" -out "${CLUSTER1_INTERMEDIATE_CA_CRT_FILE}" \
-extfile <(cat <(printf "basicConstraints = critical, CA:true, pathlen:0"))

echo "Create the certificate chain for cluster1"
export CLUSTER1_INTERMEDIATE_CA_CHAIN_FILE="${CLUSTER1_INTERMEDIATE_CA_CN}-chain.pem"
cat "${CLUSTER1_INTERMEDIATE_CA_CRT_FILE}" "${ROOT_CA_CRT_FILE}" > "${CLUSTER1_INTERMEDIATE_CA_CHAIN_FILE}"

echo "Generate intermediate CA for cluster2"
export CLUSTER2_INTERMEDIATE_CA_CN="Cluster 2 CA"
export CLUSTER2_INTERMEDIATE_CA_ORG="Org"
export CLUSTER2_INTERMEDIATE_CA_CRT_FILE="${CLUSTER2_INTERMEDIATE_CA_CN}.crt"
export CLUSTER2_INTERMEDIATE_CA_CSR_FILE="${CLUSTER2_INTERMEDIATE_CA_CN}.csr"
export CLUSTER2_INTERMEDIATE_CA_KEY_FILE="${CLUSTER2_INTERMEDIATE_CA_CN}.key"
openssl req -out "${CLUSTER2_INTERMEDIATE_CA_CSR_FILE}" -newkey rsa:2048 -nodes -keyout "${CLUSTER2_INTERMEDIATE_CA_KEY_FILE}" -subj "/CN=${CLUSTER2_INTERMEDIATE_CA_CN}/O=${CLUSTER2_INTERMEDIATE_CA_ORG}"

echo "Sign intermediate CA for cluster2 with root CA"
openssl x509 -req -days 365 -CA "${ROOT_CA_CRT_FILE}" -CAkey "${ROOT_CA_KEY_FILE}" -set_serial 2 -in "${CLUSTER2_INTERMEDIATE_CA_CSR_FILE}" -out "${CLUSTER2_INTERMEDIATE_CA_CRT_FILE}" \
-extfile <(cat <(printf "basicConstraints = critical, CA:true, pathlen:0"))

echo "Create the certificate chain for cluster2"
export CLUSTER2_INTERMEDIATE_CA_CHAIN_FILE="${CLUSTER2_INTERMEDIATE_CA_CN}-chain.pem"
cat "${CLUSTER2_INTERMEDIATE_CA_CRT_FILE}" "${ROOT_CA_CRT_FILE}" > "${CLUSTER2_INTERMEDIATE_CA_CHAIN_FILE}"

echo "Generate secret for cluster1"
kubectl create secret generic cacerts -n istio-system \
   --from-file=ca-cert.pem="${CLUSTER2_INTERMEDIATE_CA_CRT_FILE}" \
   --from-file=ca-key.pem="${CLUSTER2_INTERMEDIATE_CA_KEY_FILE}" \
   --from-file=cert-chain.pem="${CLUSTER2_INTERMEDIATE_CA_CHAIN_FILE}" \
   --from-file=root-cert.pem="${ROOT_CA_CRT_FILE}" -oyaml --dry-run=client > server-cluster-secret.yaml

echo "Generate secret for cluster2"
kubectl create secret generic cacerts -n istio-system \
   --from-file=ca-cert.pem="${CLUSTER1_INTERMEDIATE_CA_CRT_FILE}" \
   --from-file=ca-key.pem="${CLUSTER1_INTERMEDIATE_CA_KEY_FILE}" \
   --from-file=cert-chain.pem="${CLUSTER1_INTERMEDIATE_CA_CHAIN_FILE}" \
   --from-file=root-cert.pem="${ROOT_CA_CRT_FILE}" -oyaml --dry-run=client > client-cluster-secret.yaml

echo "Cleanup intermediate files beside the secrets"
rm -f "${ROOT_CA_KEY_FILE}" "${ROOT_CA_CRT_FILE}" \
   "${CLUSTER1_INTERMEDIATE_CA_CRT_FILE}" "${CLUSTER1_INTERMEDIATE_CA_CSR_FILE}" "${CLUSTER1_INTERMEDIATE_CA_KEY_FILE}" "${CLUSTER1_INTERMEDIATE_CA_CHAIN_FILE}" \
   "${CLUSTER2_INTERMEDIATE_CA_CRT_FILE}" "${CLUSTER2_INTERMEDIATE_CA_CSR_FILE}" "${CLUSTER2_INTERMEDIATE_CA_KEY_FILE}" "${CLUSTER2_INTERMEDIATE_CA_CHAIN_FILE}"
