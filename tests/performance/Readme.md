# Istio Performance Tests

This directory contains the scripts for running kyma performance tests.

## Test Setup

You can set up your custom KYMA_DOMAIN exporting `KYMA_DOMAIN` environment variable. By default the variable will be set with your current Kubeconfig domain.

1. Deploy a Kubernetes cluster with Istio module
2. Run `make test-performance`

## Test Results

Running the test will result in two reports:

- `summary-no-sidecar` for running requests with 500 virtual users to a workload with no Istio proxy
- `summary-sidecar` for running requests with 500 virtual users to a workload with Istio proxy injected

## Access Grafana

Grafana is available under <https://grafana.KYMA_DOMAIN>. Password is stored in `default/load-testing-grafana` secret.

## Scale Istio Ingressgateway

```
kubectl patch -n kyma-system istios.operator.kyma-project.io default --type merge --patch "$(cat <<EOF
spec:
    components:
        ingressGateway:
            k8s:
                hpaSpec:
                    maxReplicas: 10
                    minReplicas: 10
EOF
)"
```
