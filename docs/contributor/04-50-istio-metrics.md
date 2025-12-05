# Istio CR Metrics

## Overview

Istio operator metrics provides metrics about the configuration status of Istio custom resources (CRs) deployed in the cluster. 
These metrics help monitor the health and status of Kyma Istio managed Istio installations.

## Metrics

The operator provides metrics that are defined by the controller-runtime by default, as well as custom metrics specific to Istio CR configurations.
The following custom metrics are available:

| Metric Name                                        | Description                                                                                                 |
|----------------------------------------------------|-------------------------------------------------------------------------------------------------------------|
| **istio_ext_auth_providers_total**                     | Specifies the total number of external authorization providers defined in the Istio CR.                                   |
| **istio_ext_auth_timeout_configured_number_total**     | Specifies the total number of external authorization providers with timeout configured in the Istio CR.                   |
| **istio_ext_auth_path_prefix_configured_number_total** | Specifies the total number of external authorization providers with path prefix configured in the Istio CR.               |
| **istio_num_trusted_proxies_configured**               | Indicates whether **numTrustedProxies** is configured in the Istio CR (`1` for configured, `0` for not configured). |
| **istio_prometheus_merge_enabled**                     | Indicates whether Prometheus merge is enabled in the Istio CR (`1` for enabled, `0` for disabled).              |
| **istio_compatibility_mode_enabled**                   | Indicates whether compatibility mode is enabled in the Istio CR (`1` for enabled, `0` for disabled).            |
| istio_egress_gateway_used                          | Indicates whether the egress gateway is used in the Istio CR (`1` for used, `0` for not used).                  |
