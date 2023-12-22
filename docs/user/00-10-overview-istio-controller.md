# Istio Controller

## Overview

Istio Controller is part of Kyma Istio Operator. Its role is to manage the installation of Istio as defined by the Istio custom resource (CR). The controller is responsible for:
- Installing, upgrading, and uninstalling Istio
- Restarting workloads that have a proxy sidecar to ensure that these workloads are using the correct Istio version.

## Istio Version

The version of Istio is dependent on the version of Istio Controller that you use. This means that if a new version of Istio Controller introduces a new version of Istio, deploying the controller will automatically trigger an upgrade of Istio.

## Istio Custom Resource

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation. To learn more, read the [Istio CR documentation](04-00-istio-custom-resource.md).

## Restart of Workloads with Enabled Sidecar Injection

When the Istio version is updated or the configuration of the proxies is changed, the Pods that have Istio injection enabled are automatically restarted. This is possible for all resources that allow for a rolling restart. If Istio is uninstalled, the workloads are restarted again to remove the sidecars.
However, if a resource is a job, a ReplicaSet that is not managed by any deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In such cases, a warning is logged, and you must manually restart the resources.

## Status Codes

|     Code     | Description                                  |
|:------------:|:---------------------------------------------|
|   `Ready`    | Controller finished reconciliation.          |
| `Processing` | Controller is installing or upgrading Istio. |
|  `Deleting`  | Controller is uninstalling Istio.            |
|   `Error`    | An error occurred during reconciliation.     |
|  `Warning`   | Controller is misconfigured.                 |

## X-Forwarded-For HTTP Header

The **X-Forwarded-For** (XFF) header is only supported on AWS clusters.

The X-Forwarded-For header conveys the client IP address and the chain of intermediary proxies traversed by the request to reach the Istio Service Mesh.
The header might not include all IP addresses if an intermediary proxy does not support modifying the header.
Due to [technical limitations of AWS Classic ELBs](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-proxy-protocol.html#proxy-protocol) when using an IPv4 connection the header does not include the public of the load balancer in front of the Istio Ingress Gateway.
Moreover, the Istio Ingress Gateway Envoy will not append the private IP address of the load balancer to the X-Forwarded-For header, effectively removing this information from the request.

## TLS termination
The `istio-ingressgateway` Service creates a Layer 4 load balancer, that does not terminate TLS connections. Within the Istio Service Mesh,
the TLS termination process is handled by the Ingress Gateway Envoy proxy, which serves as a middleman between the incoming traffic and the backend services.