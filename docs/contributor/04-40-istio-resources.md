# Kyma Istio Additional Resources

## Overview

The additional Istio resources include the Kyma configuration of Istio. They consist of:


- Configuration details for Istio monitoring containing specifications for Grafana dashboards
- Configuration for Istio Ingress Gateway, which handles incoming traffic to Kyma
- Configuration for enabling Mutual TLS (mTLS) cluster-wide in the `STRICT` mode
- Information about Istio [VirtualService](https://istio.io/docs/reference/config/networking/virtual-service/), which indicates whether Istio is operational.

## Prerequisites

Installation of Istio resources requires Kyma prerequisties, namely [`cluster essentials`](../cluster-essentials) and [`certificates`](../certificates), to be installed first.
