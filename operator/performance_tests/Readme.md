
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
helm dependency update load-testing/.
helm install --create-namespace goat-test -n load-test load-testing/.
```

- Run from main directory:
```
make perf-test
```
