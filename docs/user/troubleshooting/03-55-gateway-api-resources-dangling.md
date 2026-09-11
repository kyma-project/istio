# Deletion Blocked by Gateway API Resources

Follow the steps outlined in this troubleshooting guide if the Istio module deletion is blocked because of existing Gateway API resources on the cluster.

## Symptom

The Istio custom resource (CR) is in the `Warning` state. The condition of type **Ready** is set to `false` with the reason `GatewayAPIResourcesDangling`. To verify this, run the command:

```bash
kubectl get istio default -n kyma-system -o jsonpath='{.status.conditions[0]}'
```

You get an output similar to this one:

```bash
{"lastTransitionTime":"2024-09-26T18:23:00Z","message":"Gateway API deletion blocked because of existing Gateway API resources","reason":"GatewayAPIResourcesDangling","status":"False","type":"Ready"}
```

>### Note:
> If you intended to delete the Istio module, the symptoms described in this document are expected, and you must clean up the remaining resources yourself. To check which resources are blocking the deletion, see the logs of the `istio-controller-manager` container.

## Cause

The Istio module deletion is blocked because there are user-created Gateway API resources (for example, `HTTPRoute` or `Gateway` objects) still present on the cluster, and at least one of the Gateway API CRDs is managed by the Kyma Istio module.

The module uses a [blocking deletion strategy](https://github.com/kyma-project/community/issues/765) to prevent orphaned resources after the CRDs are removed.

## Solution

1. To identify which Gateway API resources are blocking the deletion, run:

    ```bash
    kubectl logs -n kyma-system -l app=istio-controller-manager --tail=100 | grep "Gateway API resource is blocking"
    ```

2. Remove or migrate the listed resources.

3. Once all blocking resources are removed, the Istio module resumes deletion automatically.

Alternatively, if you want to force the deletion without removing the resources (the CRDs and the Gateway API objects will remain on the cluster), you can remove the finalizer from the Istio CR:

1. To edit the Istio CR, run:

    ```bash
    kubectl edit istio -n kyma-system default
    ```

2. Delete the following lines:

    ```yaml
    finalizers:
    - istios.operator.kyma-project.io/istio-installation
    ```

3. Save the changes.

When the finalizer is removed, the Istio module is deleted. The Gateway API CRDs and any existing Gateway API resources remain on the cluster.
