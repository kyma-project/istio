# Istio Module

## What Is Istio?

Istio is an open-source service mesh that provides a uniform way to manage, connect, and secure microservices. It helps to manage traffic, enhance security capabilities, and provide telemetry data for understanding service behavior. To learn more, read the [Istio documentation](https://istio.io/latest/).

## Kyma Istio Operator

Kyma Istio Operator is an essential part of the Istio module that handles the management and configuration of the Istio service mesh. It contains [Istio Controller](./00-10-overview-istio-controller.md) that is responsible for installing, uninstalling, and upgrading Istio.

The latest release includes the following versions of Istio and Envoy:  

**Istio version:** 1.23.2

**Envoy version:** 1.31.2

> [!NOTE]
> If you want to enable compatibility with the previous minor version of Istio, see [Compatibility Mode](https://kyma-project.io/#/istio/user/00-10-overview-istio-controller?id=compatibility-mode).

## Useful Links

To gain a better understanding of the Istio module's capabilities, see the overview of:
- [Istio Controller](./00-10-overview-istio-controller.md)
- [Istio Service Mesh](./00-20-overview-service-mesh.md)
- [Istio Sidecars](./00-30-overview-istio-sidecars.md)
- [Default Istio Setup](./00-40-overview-istio-setup.md)
- [Default Resources and Autoscaling Configuration](./00-50-resource-configuration.md)

To learn how to use the Istio module, follow the [tutorials](./tutorials/) and [operation guides](./operation-guides/). For more in-depth information, read [Istio Custom Resource specification](./04-00-istio-custom-resource.md) and [technical reference](./technical-reference/) documentation. If you face any problems, refer to the [troubleshooting guides](./troubleshooting/) for assistance.

If you are interested in the detailed documentation of Kyma Istio Operator's design and technical aspects, check the [contributor](https://github.com/kyma-project/istio/tree/main/docs/contributor) directory.
