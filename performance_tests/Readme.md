# Istio performance tests

This directory contains the scripts for running kyma performance tests.

## Test Setup

1. Deploy a Kubernetes cluster with Kyma on a production profile
2. Run `make test-performance`

## Test results

Running the test will result in two reports:

- `summary-no-sidecar` for workload with no Istio proxy
- `summary-sidecar` for workload with Istio proxy injected

## Access grafana

Grafana is available under <https://grafana.KYMA_DOMAIN>. Password is stored in `default/load-testing-grafana` secret.
