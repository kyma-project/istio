# Istio module

## What is Istio

<img src="./docs/assets/istio-whitelogo-bluebackground-framed.svg" alt="Istio logo" style="height: 100px; width:100px;"/>

Istio is an open-source service mesh that provides a uniform way to manage, connect, and secure microservices. It helps to manage traffic, enhance security capabilities, and provide telemetry data for understanding service behavior. Read the [Istio documentation](https://istio.io/latest/) to learn more.

## Istio module

The Istio module allows you to add Istio Operator to the Kyma runtime. Within Istio Operator, Istio Controller is responsible for installing, uninstalling, and managing Istio. For more information, read the [Istio Controller documentation](/docs/user/00-10-overview-istio-controller.md).

## Install Istio Operator and Istio from the latest release

### Prerequisites

- Access to a Kubernetes (v1.24 or higher) cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Procedure

1. To install Istio, you must install the latest version of Istio Operator and Istio CRD first. Run:

   ```bash
   kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml
   ```

2. To get Istio installed, apply the default Istio CR:

   ```bash
   kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml
   ```

   You should get a result similar to this example:

   ```bash
   istio.operator.kyma-project.io/default created
   ```

3. Check the state of Istio CR to verify if Istio was installed successfully:

   ```bash
   kubectl get -n kyma-system istios/default
   ```

   After successful installation, you get the following output:

   ```bash
   NAME      STATE
   default   Ready
   ```

For more installation options, visit [Install the Istio module](/docs/contributor/01-00-installation.md).

## Documentation

To learn how to use the Istio module, read the documentation in the [user](/docs/user) directory.

If you are interested in the detailed documentation of the module's design and technical aspects, check the [contributor](/docs/contributor/) directory.

## Contributing

To contribute to this project, follow the general [Kyma project contributing](https://github.com/kyma-project/community/blob/main/docs/contributing/02-contributing.md) guidelines.