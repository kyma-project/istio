# Kyma Istio Additional Resources

## Overview

The additional Istio resources include the Kyma configuration of Istio. They consist of:

- Istio monitoring configuration details providing Grafana dashboards specification
- Istio Ingress Gateway configuring incoming traffic to Kyma
- Mutual TLS (mTLS) configuration enabling mTLS cluster-wide in the STRICT mode
- Istio [VirtualService](https://istio.io/docs/reference/config/networking/virtual-service/) informing whether Istio is up and running

## Prerequisites

Installation of Istio resources requires Kyma prerequisties, namely [`cluster essentials`](../cluster-essentials) and [`certificates`](../certificates), to be installed first.
