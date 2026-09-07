[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/istio)](https://api.reuse.software/info/github.com/kyma-project/istio)
# Istio

## What is Istio

Istio is an open-source service mesh that provides a uniform way to manage, connect, and secure microservices. It helps to manage traffic, enhance security capabilities, and provide telemetry data for understanding service behavior. See the [open-source Istio documentation](https://istio.io/latest/docs/).

The Istio module installs and manages Istio in your Kyma cluster. The version of Istio depends on the version of the Istio module that you use. When a new version of the Istio module introduces a new version of Istio, an upgrade of the module causes an automatic upgrade of Istio. To track the changes introduced in open-source Istio, learn which version of Istio the latest version of the Istio module installs. For this information, follow [Releases](https://github.com/kyma-project/istio/releases). To learn how to enable compatibility with the previous minor version of Istio, see [Compatibility Mode](./docs/user/00-10-istio-version.md#compatibility-mode).

## Install the Latest Release of the Istio Module

### Prerequisites

- Access to a Kubernetes cluster
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

## Repository Conventions

This repository uses two metadata files that drive CI/CD automation. They should not be changed by developers working on forks unless explicitly noted.

### `MINOR_VERSION`

Contains the current major.minor version of the module (e.g. `1.30`). It represents what this branch *is*, not what it will become:

- On `main`: the version currently in development (e.g. `1.30` means the next release will be `1.30.x`)
- On a `release-X.Y` branch: always `X.Y`, set when the branch was created and never changed
- On feature/bugfix branches: inherited from the branch they were cut from — no changes needed

When a new minor release is prepared, the `prepare-new-minor` workflow branches off `release-X.Y`, then bumps `MINOR_VERSION` on `main` to the next minor via a PR.

### `RELEASE_REPOSITORY`

Contains the canonical GitHub repository where official module releases are published (e.g. `kyma-project/istio`). It is used by:

- `hack/ci/get-reference-release.sh` — to find the official release to upgrade from in upgrade tests
- `make deploy-release` — to install a specific release version into a cluster

**Forks should not change this file.** A developer working on a fork still upgrades from and installs the official upstream releases. Only the repository that owns the official release process should have its own name here.

## `hack/` Directory

The `hack/` directory contains scripts and supporting files used during development and CI/CD. Scripts directly under `hack/` are general-purpose developer utilities that can also be useful outside of CI. Scripts under `hack/ci/` are intended to be run exclusively by the CI environment and typically require CI-specific credentials and environment variables.

### Named Configurations

Both `hack/ci/k3d/` and `hack/ci/gardener/` use a **named configuration** convention to select a test environment preset. Each preset lives in a subdirectory:

```
hack/ci/k3d/configurations/<name>/vars.sh
hack/ci/gardener/configurations/<name>/vars.sh
hack/ci/gardener/configurations/<name>/shoot.yaml
```

The configuration is selected by setting `K3D_CONFIGURATION` or `GARDENER_CONFIGURATION` to the preset name (e.g. `default`, `gcp-ipv4`, `aws-ipv4`). The `vars.sh` file defines all environment-specific variables for that preset and they are auto-exported into the shell. For Gardener configurations, `shoot.yaml` is a cluster template whose `$VAR` references are filled in from those variables via `envsubst`.

This keeps environment differences (cloud provider, IP stack, node settings, etc.) entirely inside the configuration directory, while the scripts themselves stay generic.

### `common.sh`

All `hack/ci/` scripts source `hack/ci/common.sh` at startup. It provides shared utilities used across all CI scripts — argument and environment variable validation, configuration loading, and structured log output.

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
