[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/istio)](https://api.reuse.software/info/github.com/kyma-project/istio)
# Istio

## What is Istio

<img src="/docs/assets/istio-whitelogo-bluebackground-framed.svg" alt="Istio logo" style="height: 100px; width:100px;"/>

Istio is an open-source service mesh that provides a uniform way to manage, connect, and secure microservices. It helps to manage traffic, enhance security capabilities, and provide telemetry data for understanding service behavior. Read the [Istio documentation](https://istio.io/latest/) to learn more.

## Kyma Istio Operator

Kyma Istio Operator is a component of the Kyma runtime that handles the management and configuration of the Istio service mesh. Within Kyma Istio Operator, [Istio Controller](/docs/user/00-10-overview-istio-controller.md) is responsible for installing, uninstalling, and upgrading Istio.

The latest release includes the following versions of Istio and Envoy:
**Istio version:** 1.21.3
**Envoy version:** 1.29.5

If you want to enable compatibility with the previous minor version of Istio, see [Compatibility Mode](https://kyma-project.io/#/istio/user/00-10-overview-istio-controller?id=compatibility-mode).

## Install Kyma Istio Operator and Istio from the latest release

### Prerequisites

- Access to a Kubernetes (v1.24 or higher) cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Procedure

1. Create the `kyma-system` namespace and label it with `istio-injection=enabled`:

   ```bash
   kubectl create namespace kyma-system
   kubectl label namespace kyma-system istio-injection=enabled --overwrite
   ```

2. To install Istio, you must install the latest version of Kyma Istio Operator and Istio CRD first.
   In order to install the standard version, run :
   ```bash
   kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml
   ```

   In order to install the experimental version, run :
   ```bash
   kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager-experimental.yaml
   ```
   
3. To get Istio installed, apply the default Istio CR:

   ```bash
   kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml
   ```

   You should get a result similar to this example:

   ```bash
   istio.operator.kyma-project.io/default created
   ```

4. Check the state of Istio CR to verify if Istio was installed successfully:

   ```bash
   kubectl get -n kyma-system istios/default
   ```

   After successful installation, you get the following output:

   ```bash
   NAME      STATE
   default   Ready
   ```

For more installation options, visit [the installation guide](/docs/contributor/01-00-installation.md).

## Useful links

To learn how to use Kyma Istio Operator, read the documentation in the [`user`](/docs/user) directory.

If you are interested in the detailed documentation of the Kyma Istio Operator's design and technical aspects, check the [`contributor`](/docs/contributor) directory.

## Contributing
<!--- mandatory section - do not change this! --->

See the [Contributing](CONTRIBUTING.md) guidelines.

## Code of Conduct
<!--- mandatory section - do not change this! --->

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing
<!--- mandatory section - do not change this! --->

See the [license](./LICENSE) file.
