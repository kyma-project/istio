# Istio Controller

## Overview

Istio Controller is part of Kyma Istio Operator. Its role is to manage the installation of Istio as defined by the Istio custom resource (CR). The controller is responsible for:
- Installing, upgrading, and uninstalling Istio
- Restarting workloads that have a proxy sidecar to ensure that these workloads are using the correct Istio version.

## Istio Version

The version of Istio is dependent on the version of Istio Controller that you use. This means that if a new version of Istio Controller introduces a new version of Istio, deploying the controller will automatically trigger an upgrade of Istio.

## Upgrades and Downgrades

You can only skip a version of Kyma Istio Operator if the difference between the minor version of Istio it contains and the minor version of Istio you're using is not greater than one (for example, 1.2.3 -> 1.3.0).
If the difference is greater than one minor version (for example, 1.2.3 -> 1.4.0), the reconciliation fails.
The same happens if you try to update the major version (for example, 1.2.3 -> 2.0.0) or downgrade the version. Such scenarios are not supported.

## Istio Custom Resource

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation. To learn more, read the [Istio CR documentation](04-00-istio-custom-resource.md).

## Restart of Workloads with Enabled Sidecar Injection

When the Istio version is updated or the configuration of the proxies is changed, the Pods that have Istio injection enabled are automatically restarted. This is possible for all resources that allow for a rolling restart. If Istio is uninstalled, the workloads are restarted again to remove the sidecars.
However, if a resource is a job, a ReplicaSet that is not managed by any deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In such cases, a warning is logged, and you must manually restart the resources.
Istio Operator does not restart an Istio sidecar proxy, if it has a custom image set. See [Resource Annotations](https://istio.io/latest/docs/reference/config/annotations/#SidecarProxyImage).

## Status Codes

|     Code     | Description                                  |
|:------------:|:---------------------------------------------|
|   `Ready`    | Controller finished reconciliation.          |
| `Processing` | Controller is installing or upgrading Istio. |
|  `Deleting`  | Controller is uninstalling Istio.            |
|   `Error`    | An error occurred during reconciliation.     |
|  `Warning`   | Controller is misconfigured.                 |

Conditions:

| CR state   | Type                         | Status | Reason                            | Message                                                                                  |
|------------|------------------------------|--------|-----------------------------------|------------------------------------------------------------------------------------------|
| Ready      | Ready                        | True   | ReconcileSucceeded                | Reconciliation succeeded                                                                 |
| Error      | Ready                        | False  | ReconcileFailed                   | Reconciliation failed                                                                    |
| Warning    | Ready                        | False  | OlderCRExists                     | This Istio custom resource is not the oldest one and does not represent the module state |
| Processing | Ready                        | False  | IstioInstallNotNeeded             | Istio installation is not needed                                                         |
| Processing | Ready                        | False  | IstioInstallSucceeded             | Istio installation succeeded                                                             |
| Processing | Ready                        | False  | IstioUninstallSucceeded           | Istio uninstallation succeded                                                            |
| Error      | Ready                        | False  | IstioInstallUninstallFailed       | Istio install or uninstall failed                                                        |
| Error      | Ready                        | False  | IstioCustomResourceMisconfigured  | Istio custom resource has invalid configuration                                          |
| Warning    | Ready                        | False  | IstioCustomResourcesDangling      | Istio deletion blocked because of existing Istio custom resources                        |
| Processing | Ready                        | False  | CustomResourcesReconcileSucceeded | Custom resources reconciliation succeeded                                                |
| Error      | Ready                        | False  | CustomResourcesReconcileFailed    | Custom resources reconciliation failed                                                   |
| Processing | ProxySidecarRestartSucceeded | True   | ProxySidecarRestartSucceeded      | Proxy sidecar restart succeeded                                                          |
| Error      | ProxySidecarRestartSucceeded | False  | ProxySidecarRestartFailed         | Proxy sidecar restart failed                                                             |
| Warning    | ProxySidecarRestartSucceeded | False  | ProxySidecarManualRestartRequired | Proxy sidecar manual restart is required for some workloads                              |
| Processing | Ready                        | False  | IngressGatewayReconcileSucceeded  | Istio Ingress Gateway reconciliation succeeded                                           |
| Error      | Ready                        | False  | IngressGatewayReconcileFailed     | Istio Ingress Gateway reconciliation failed                                              |

## X-Forwarded-For HTTP Header

The XFF header conveys the client IP address and the chain of intermediary proxies that the request traversed to reach the Istio service mesh.
The header might not include all IP addresses if an intermediary proxy does not support modifying the header.
Due to [technical limitations of AWS Classic ELBs](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-proxy-protocol.html#proxy-protocol), when using an IPv4 connection, the header does not include the public IP of the load balancer in front of Istio Ingress Gateway.
Moreover, Istio Ingress Gateway Envoy does not append the private IP address of the load balancer to the XFF header, effectively removing this information from the request.

## TLS termination
The `istio-ingressgateway` Service creates a Layer 4 load balancer, that does not terminate TLS connections. Within the Istio service mesh,
the TLS termination process is handled by the Ingress Gateway Envoy proxy, which serves as a middleman between the incoming traffic and the backend services.

## Labeling Resources

In accordance with the decision [Consistent Labeling of Kyma Modules](https://github.com/kyma-project/community/issues/864), the Istio Operator resources use the standard Kubernetes labels:


```yaml
kyma-project.io/module: istio
app.kubernetes.io/name: istio-operator
app.kubernetes.io/instance: istio-operator-default
app.kubernetes.io/version: "x.x.x"
app.kubernetes.io/component: operator
app.kubernetes.io/part-of: istio
```

All other resources, such as the external `istio` component and its respective resources, use only the Kyma module label:

```yaml
kyma-project.io/module: istio
```

Run this command to get all resources created by the Istio module:

```bash
kubectl get all|<resources-kind> -A -l kyma-project.io/module=istio
```

## Compatibility Mode

To enable compatibility mode in the Istio module, you can set the **spec.compatibilityMode** field in the Istio CR. This allows you to mitigate breaking changes when a new release introduces an Istio upgrade. The Istio module applies an opinionated subset of Istio compatibilityVersion, and supports compatibility with the previous minor version of Istio. For example, the Istio module with Istio 1.21.0 applies a compatibility version of Istio 1.20. For more information, see [Compatibility Versions](https://istio.io/latest/docs/setup/additional-setup/compatibility-versions/).
