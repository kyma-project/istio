# Install the Istio module

- [Install the Istio module](#install-the-istio-module)
  - [Prerequisites](#prerequisites)
  - [Install Istio Operator manually](#install-istio-operator-manually)
    - [Use Istio Operator to install or uninstall Istio](#use-istio-operator-to-install-or-uninstall-istio)
  - [Install Istio in modular Kyma on a local k3d cluster](#install-istio-in-modular-kyma-on-a-local-k3d-cluster)
    - [Use Lifecycle Manager to install Istio in modular Kyma on k3d](#use-lifecycle-manager-to-install-istio-in-modular-kyma-on-k3d)
  - [Install with artifacts built for the `main` branch of the Istio repository](#install-with-artifacts-built-for-the-main-branch-of-the-istio-repository)

## Prerequisites

- Access to a Kubernetes cluster (you can use [k3d](https://k3d.io/v5.5.1/))
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)
- [Docker](https://www.docker.com)
- [Kyma CLI](https://kyma-project.io/docs/kyma/latest/04-operation-guides/operations/01-install-kyma-CLI)

## Install Istio Operator manually

1. Clone the project.

```bash
git clone https://github.com/kyma-project/istio.git && cd istio
```

2. Set the Istio Operator image name.

```bash
export IMG=istio-operator:0.0.1
export K3D_CLUSTER_NAME=kyma
```

3. Provision k3d cluster.

```bash
kyma provision k3d
```
>**TIP:** To verify the correctness of the project, build it using the `make build` command.

4. Build the image.

```bash
make docker-build
```

5. Push the image to the registry.

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

6. Deploy Isito Operator.

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

## Install Istio in modular Kyma on a local k3d cluster

1. Set up a local k3d cluster and a local Docker registry.

```bash
k3d cluster create kyma --registry-create k3d-kyma-registry:0.0.0.0:5001
```

2. Add the `etc/hosts` entry to register the local Docker registry under the name `k3d-kyma-registry`.

```bash
127.0.0.1 registry.localhost
```

3. Export environment variables pointing to the module and the module's image registries.

```bash
export IMG_REGISTRY=registry.localhost:5001/unsigned/operator-images
export MODULE_REGISTRY=registry.localhost:5001/unsigned
```

4. Build the Istio module.

```bash
make module-build
```

This command builds an OCI image for the Istio module and pushes it to the registry and path, as defined in `MODULE_REGISTRY`.

5. Build Istio Operator's image.

```bash
make module-image
```

This command builds a Docker image for Istio Operator and pushes it to the registry and path, as defined in `IMG_REGISTRY`.

6. Verify that the Istio module's image and Istio Operator's image are pushed to the local registry. Run:

```bash
curl registry.localhost:5001/v2/_catalog
```
If the images were pushed successfully, you get the following output:

```json
{
    "repositories": [
        "unsigned/component-descriptors/kyma.project.io/module/istio",
        "unsigned/operator-images/istio-operator"
    ]
}
```

7. Inspect the generated module template. Change the existing repository context in `spec.descriptor.component`:

```yaml
repositoryContexts:                                                                           
- baseUrl: registry.localhost:5000/unsigned                                                   
  componentNameMapping: urlPath                                                               
  type: ociRegistry
```
>**NOTE:** Because Pods inside the k3d cluster use the docker-internal port of the registry, it tries to resolve the registry against port 5000 instead of 5001. k3d has registry aliases but module-manager is not part of k3d and thus does not know how to properly alias `registry.localhost:5001`

>**TIP:** Apply `"operator.kyma-project.io/use-local-template": "true"` to make sure that Lifecycle Manager uses the registry URL present in the ModuleTemplate.

### Use Lifecycle Manager to install Istio in modular Kyma on k3d

1. Install the latest version of `lifecycle-manager`.

```bash
kyma alpha deploy
```
If deployed successfully, you get the following output:

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

Kyma installation is ready, but no module is activated yet.

```bash
kubectl get kymas.operator.kyma-project.io -A
NAMESPACE    NAME           STATE   AGE
kcp-system   default-kyma   Ready   71s
```

2. Apply `template.yaml` to register Istio as a module known for Lifecycle Manager.

Istio Module is a known module, but it is not activated.

```bash
kubectl get moduletemplates.operator.kyma-project.io -A 
NAMESPACE    NAME                  AGE
kcp-system   moduletemplate-istio   2m24s
```

3. Give Lifecycle Manager permission to install CustomResourceDefinition (CRD) cluster-wide.

>**NOTE:** This is a temporary workaround only required in the single-cluster mode.

Module-manager must be able to apply CRDs to install modules. When it operates in remote mode, where the control-plane manages remote clusters, it receives an administrative kubeconfig that allows it to target the remote cluster and apply CRDs. However, when it operates in local mode (single-cluster mode), it uses Service Account and does not have permission to create CRDs by default.

Run the following command to make sure the module manager's Service Account is granted an administrative role:

```bash
kubectl edit clusterrole lifecycle-manager-manager-role
```

Add the following configuration:

```yaml
- apiGroups:                                                                                                                  
  - "*"                                                                                                                       
  resources:                                                                                                               
  - "*"                                                                                                                       
  verbs:                                                                                                                      
  - "*"
```

4. Enable the Istio module.

```bash
kyma alpha enable module istio -c alpha
```

## Install with artifacts built for the `main` branch of the Istio repository

You can install the Istio module using the artifacts that are created by `post-istio-module-build` job. To do so, follow these steps:

1. Install Lifecycle Manager in the target cluster.
   
```bash
kyma alpha deploy
```
2. Deploy the ModuleTemplate generated by the job. You can find it in the job artifacts.
3. Install the Istio module. 
   
```bash
kyma alpha enable module istio -c alpha
```
