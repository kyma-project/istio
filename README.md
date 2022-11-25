# Istio Manager

<img src="https://istio.io/latest/img/istio-whitelogo-bluebackground-framed.svg" alt="Istio logo" style="height: 100px; width:100px;"/>

## Overview

Istio Manager is a module compatible with Lifecycle Manager that allows you to add Kyma Istio Operator to the Kyma ecosystem.

See also:

- [lifecycle-manager documentation](https://github.com/kyma-project/lifecycle-manager)
- [Istio documentation](https://https://istio.io/latest/)

## Prerequisites

- Access to a k8s cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

```bash
# you could use one of the following options

# option 1: using brew
brew install kubebuilder

# option 2: fetch sources directly
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```

## Manual `istio-operator` installation

1. Clone project

```bash
git clone https://github.com/kyma-project/istio.git && cd istio/operator
```

2. Set `istio-operator` image name

```bash
export IMG=istio-operator:0.0.1
export K3D_CLUSTER_NAME=kyma
```

3. Provision k3d cluster with `kyma provision k3d`

4. Build project

```bash
make build
```

5. Build image

```bash
make docker-build
```

6. Push image to registry

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

7. Deploy

```bash
make deploy
```

## Using `istio-operator`

- Install Istio on cluster

```bash
kubectl apply -f config/samples/operator_v1alpha1_istio.yaml
```

- Delete Istio from cluster

```bash
kubectl delete -f config/samples/operator_v1alpha1_istio.yaml
```

## Installation in modular Kyma on the local k3d cluster

1. Setup local k3d cluster and local Docker registry

```bash
k3d cluster create kyma --registry-create registry.localhost:0.0.0.0:5001
```

2. Add the `etc/hosts` entry to register the local Docker registry under the `registry.localhost` name

```bash
127.0.0.1 registry.localhost
```

3. Export environment variables (ENVs) pointing to module and the module image registries

```bash
export IMG_REGISTRY=registry.localhost:5001/unsigned/operator-images
export MODULE_REGISTRY=registry.localhost:5001/unsigned
```

4. Build Istio module

```bash
make module-build
```

This builds an OCI image for Istio module and pushes it to the registry and path, as defined in `MODULE_REGISTRY`.

5. Build Istio manager image

```bash
make module-image
```

This builds a Docker image for Istio Manager and pushes it to the registry and path, as defined in `IMG_REGISTRY`.

6. Verify if the module and the manager's image are pushed to the local registry

```bash
curl registry.localhost:5001/v2/_catalog
```

```json
{
    "repositories": [
        "unsigned/component-descriptors/kyma.project.io/module/istio",
        "unsigned/operator-images/istio-operator"
    ]
}
```

7. Inspect the generated module template

The following are temporary workarounds.

Edit the `template.yaml` file and:

- change `target` to `control-plane`

```yaml
spec:
  target: control-plane
```

>**NOTE:** This is only required in the single cluster mode

- change the existing repository context in `spec.descriptor.component`:

>**NOTE:** Because Pods inside the k3d cluster use the docker-internal port of the registry, it tries to resolve the registry against port 5000 instead of 5001. K3d has registry aliases but module-manager is not part of k3d and thus does not know how to properly alias `registry.localhost:5001`

```yaml
repositoryContexts:                                                                           
- baseUrl: registry.localhost:5000/unsigned                                                   
  componentNameMapping: urlPath                                                               
  type: ociRegistry
```

## Install modular Kyma on the k3d cluster

1. Install the latest versions of `module-manager` and `lifecycle-manager` with `kyma alpha deploy`

```bash
- Kustomize ready
- Lifecycle Manager deployed
- Module Manager deployed
- Modules deployed
- Kyma CR deployed
- Kyma deployed successfully!

Kyma is installed in version:
Kyma installation took: 18 seconds

Happy Kyma-ing! :)
```

Kyma installation is ready, but no module is activated yet

```bash
kubectl get kymas.operator.kyma-project.io -A
NAMESPACE    NAME           STATE   AGE
kcp-system   default-kyma   Ready   71s
```

2. Apply `template.yaml` to register Istio as a module known for `module-manager`

Istio Module is a known module, but not activated

```bash
kubectl get moduletemplates.operator.kyma-project.io -A 
NAMESPACE    NAME                  AGE
kcp-system   moduletemplate-istio   2m24s
```

3. Give Module Manager permission to install CustomResourceDefinition (CRD) cluster-wide

>**NOTE:** This is a temporary workaround and is only required in the single-cluster mode

Module-manager must be able to apply CRDs to install modules. In the remote mode (with control-plane managing remote clusters) it gets an administrative kubeconfig, targeting the remote cluster to do so. But in local mode (single-cluster mode), it uses Service Account and does not have permission to create CRDs by default.

Run the following to make sure the module manager's Service Account becomes an administrative role:

```bash
kubectl edit clusterrole module-manager-manager-role
```

add

```yaml
- apiGroups:                                                                                                                  
  - "*"                                                                                                                       
  resources:                                                                                                               
  - "*"                                                                                                                       
  verbs:                                                                                                                      
  - "*"
```

4. Enable Istio in Kyma

```bash
kubectl apply -f kyma.yaml -n kcp-system
```

