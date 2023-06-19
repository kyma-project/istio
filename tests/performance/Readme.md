# Istio performance tests

This directory contains the scripts for running kyma performance tests.

## Test Setup

You can set up your custom KYMA_DOMAIN exporting `KYMA_DOMAIN` environment variable. By default the variable will be set with your current Kubeconfig domain.

1. Deploy a Kubernetes cluster with Kyma on a production profile
2. Run `make test-performance`

## Test results

Running the test will result in two reports:

- `summary-no-sidecar` for running requests with 500 virtual users to a workload with no Istio proxy
- `summary-sidecar` for running requests with 500 virtual users to a workload with Istio proxy injected

## Access grafana

Grafana is available under <https://grafana.KYMA_DOMAIN>. Password is stored in `default/load-testing-grafana` secret.
