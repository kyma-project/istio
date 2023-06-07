## Install in modular Kyma on the local k3d cluster

1. Set up a local k3d cluster and a local Docker registry.

```bash
k3d cluster create kyma --registry-create registry.localhost:0.0.0.0:5001
```

2. Add the `etc/hosts` entry to register the local Docker registry under the name `registry.localhost`.

```bash
127.0.0.1 registry.localhost
```

3. Export environment variables (ENVs) pointing to the module and the module's image registries.

```bash
export IMG_REGISTRY=registry.localhost:5001/unsigned/operator-images
export MODULE_REGISTRY=registry.localhost:5001/unsigned
```

4. Build the Istio module.

```bash
make module-build
```

This command builds an OCI image for the Istio module and pushes it to the registry and path, as defined in `MODULE_REGISTRY`.

5. Build Istio Manager's image.

```bash
make module-image
```

This command builds a Docker image for Istio Manager and pushes it to the registry and path, as defined in `IMG_REGISTRY`.

1. Verify if the Istio module's image and Istio Manager's image are pushed to the local registry.

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

7. Inspect the generated module template.

>**NOTE:** The following instructions are temporary workarounds.

Edit the `template.yaml` file.

- Change `target` to `control-plane`.

```yaml
spec:
  target: control-plane
```

>**NOTE:** This is only required in the single cluster mode

- Change the existing repository context in `spec.descriptor.component`:

```yaml
repositoryContexts:                                                                           
- baseUrl: registry.localhost:5000/unsigned                                                   
  componentNameMapping: urlPath                                                               
  type: ociRegistry
```
>**NOTE:** Because Pods inside the k3d cluster use the docker-internal port of the registry, it tries to resolve the registry against port 5000 instead of 5001. K3d has registry aliases but module-manager is not part of k3d and thus does not know how to properly alias `registry.localhost:5001`

>**NOTE** Apply `"operator.kyma-project.io/use-local-template": "true"` to make sure that Lifecycle Manager will use the registry URL present in the ModuleTemplate.

## Install modular Kyma on the k3d cluster

1. Install the latest version `lifecycle-manager`.

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

>**NOTE:** This is a temporary workaround only required in the single-cluster mode

Module-manager must be able to apply CRDs to install modules. In the remote mode (with control-plane managing remote clusters) it gets an administrative kubeconfig, targeting the remote cluster to do so. But in local mode (single-cluster mode), it uses Service Account and does not have permission to create CRDs by default.

Run the following to make sure the module manager's Service Account is granted an administrative role:

```bash
kubectl edit clusterrole lifecycle-manager-manager-role
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

1. Enable Istio in Kyma

```bash
kyma alpha enable module istio
```

## Installation with artifacts built for the `main` branch of Istio repository

You can install Istio module using the artificats that are created by `post-istio-module-build` job. To do sa, follow this steps:

1. Install Lifecycle Manager in a target cluster.
   
```bash
kyma alpha deploy
```
2. Deploy the ModuleTemplate generated by the job. You can find it in the job artifacts.
3. Install the Istio module. 
   
```bash
`kyma alpha enable module istio -c alpha`
```
