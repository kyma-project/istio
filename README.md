[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/istio)](https://api.reuse.software/info/github.com/kyma-project/istio)
# Istio

## What is Istio

<img src="/docs/assets/istio-whitelogo-bluebackground-framed.svg" alt="Istio logo" style="height: 100px; width:100px;"/>

Istio is an open-source service mesh that provides a uniform way to manage, connect, and secure microservices. It helps to manage traffic, enhance security capabilities, and provide telemetry data for understanding service behavior. See the [open-source Istio documentation](https://istio.io/latest/docs/).

The Istio module installs and manages Istio in your Kyma cluster. The latest release includes the following versions of Istio and Envoy:

**Istio version:** 1.23.2

**Envoy version:** 1.31.2

> [!NOTE]
> If you want to enable compatibility with the previous minor version of Istio, see [Compatibility Mode](./docs/user/00-10-istio-version.md#compatibility-mode).

## Install the Latest Release of the Istio Module

### Prerequisites

- Access to a Kubernetes cluster (v1.24 or higher)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Procedure

1. Create the `kyma-system` namespace and label it with `istio-injection=enabled`:

   ```bash
   kubectl create namespace kyma-system
   kubectl label namespace kyma-system istio-injection=enabled --overwrite
   ```

2. Install the latest version of Istio Operator and Istio CustomResourceDefinition. You can install either the standard or experimental version.
   
   - To install the standard version, run:
      ```bash
      kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml
      ```

   - To install the experimental version, run:
      ```bash
      kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager-experimental.yaml
      ```

3. To install Istio, apply the default Istio custom resource (CR):

   ```bash
   kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml
   ```

4. To verify the Istio was installed successfully, check the state of the Istio CR.

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

To learn how to use the Istio module, read the documentation in the [`user`](/docs/user) directory.

If you are interested in the detailed documentation of the Istio module's design and technical aspects, check the [`contributor`](/docs/contributor) directory.

## Contributing
<!--- mandatory section - do not change this! --->

See the [Contributing](CONTRIBUTING.md) guidelines.

## Code of Conduct
<!--- mandatory section - do not change this! --->

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing
<!--- mandatory section - do not change this! --->

See the [license](./LICENSE) file.
