# Kyma endpoint returns not found error (404 status code)

## Symptom

You are accessing a Kyma endpoint at it reports 404 error.

## Cause

This might be cause by Istio gateway host conflicts. For example if you create two gateways with the same host, Istio ingress gateway will not be able to deterministically match incoming request to a Gateway, and it will result with requests getting 404 errors. The behaviour is described in https://istio.io/latest/docs/ops/common-problems/network-issues/#404-errors-occur-when-multiple-gateways-configured-with-same-tls-certificate. Keep in mind that creating `Ingress` resources using Istio as their ingress class will also create a `Gateway` entry underneath. To check the cluster configuration you can use `istioctl x internal-debug configz` command.

## Remedy

Make sure that a host matches one `Gateway` only.
