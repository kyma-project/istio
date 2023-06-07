# Istio Manager

<img src="./docs/istio-whitelogo-bluebackground-framed.svg" alt="Istio logo" style="height: 100px; width:100px;"/>

## Overview

Istio Manager is a module compatible with [Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager) that allows you to add Kyma Istio Operator to the Kyma runtime. Istio Operator is responsible for installing, uninstalling, and upgrading [Istio](https://https://istio.io/latest/). {to be checked}

## Installation

### Prerequisites

- Access to a k8s cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)
  
### Install Istio Operator manually

1. Clone the project.

```bash
git clone https://github.com/kyma-project/istio.git && cd istio
```

2. Set Istio Operator image name.

```bash
export IMG=istio-operator:0.0.1
export K3D_CLUSTER_NAME=kyma
```

3. Provision k3d cluster.

```bash
kyma provision k3d
```

4. Build the project.

```bash
make build
```

5. Build the image.

```bash
make docker-build
```

6. Push the image to the registry.

<div tabs name="Push image" group="istio-operator-installation">
  <details>
  <summary label="k3d">
  k3d
  </summary>

   ```bash
   k3d image import $IMG -c $K3D_CLUSTER_NAME
   ```

  </details>
  <details>
  <summary label="Docker registry">
  Globally available Docker registry
  </summary>

   ```bash
   make docker-push
   ```

  </details>
</div>

7. Deploy.

```bash
make deploy
```

### Use Istio Operator to install or uninstall Istio

- Install Istio in your cluster.

```bash
kubectl apply -f config/samples/operator_v1alpha1_istio.yaml
```

- Delete Istio from your cluster.

```bash
kubectl delete -f config/samples/operator_v1alpha1_istio.yaml
```

Chek more [installation options](./docs/contributor/01-00-installation.md).

## Read more

If you want to contribute to the project, see the [contributor](./docs/contributor/) directory. It contains documentation and guidelines specifically designed for developers.

## Contributing

To contribute to this project, follow the general [Kyma project contributing](https://github.com/kyma-project/community/blob/main/docs/contributing/02-contributing.md) guidelines.