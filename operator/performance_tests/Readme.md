
# Istio performance tests

This directory contains the scripts for running kyma performance tests.

## Test Setup

- Deploy Kubernetes cluster with Kyma
- Export the following variable:
```
export DOMAIN=<YOUR_CLUSTER_DOMAIN>
```
- Deploy helm chart to start load-testing

```sh
helm dependency update operator/performance_tests/load-testing/.
helm install goat-test --set domain="$DOMAIN" --create-namespace -n load-test operator/performance_tests/load-testing/.
```

- Run from main directory:
```
make perf-test
```
