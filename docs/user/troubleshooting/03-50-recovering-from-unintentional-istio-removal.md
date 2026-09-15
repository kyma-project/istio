# Istio Module Deletion Blocked

Follow the steps in this guide if the Istio module deletion is blocked because Istio or Gateway API resources still exist in the cluster.

## Symptom

The Istio custom resource (CR) is in the `Warning` state. The condition of type **Ready** is set to `false` with the reason `IstioCustomResourcesDangling` or `GatewayAPIResourcesDangling`. To verify this, run:

```bash
kubectl get istio default -n kyma-system -o jsonpath='{.status.conditions[0]}'
```

You get one of the following outputs:

```bash
{"lastTransitionTime":"2024-09-26T18:23:00Z","message":"Istio deletion blocked because of existing Istio custom resources","reason":"IstioCustomResourcesDangling","status":"False","type":"Ready"}
```

```bash
{"lastTransitionTime":"2024-09-26T18:23:00Z","message":"Gateway API deletion blocked because of existing Gateway API resources","reason":"GatewayAPIResourcesDangling","status":"False","type":"Ready"}
```

## Cause

For example, the issue occurs when you delete Istio, but there are still VirtualService or Gateway API resources (such as `HTTPRoute` or `Gateway` objects) either created by you or installed by another Kyma component or module. In such cases, the hooked finalizer pauses the deletion of Istio until you remove all the related resources. This [blocking deletion strategy](https://github.com/kyma-project/community/issues/765) is intentionally designed and is enabled by default for the Istio module.

The module uses a [blocking deletion strategy](https://github.com/kyma-project/community/issues/765) to prevent orphaned resources after the CRDs are removed.

## Solution

Choose one of the following options depending on whether you want to revert the deletion or permanently remove the Istio module.

### Revert an Accidental Deletion

If you unintentionally deleted the Istio module and want to restore the cluster to its previous state, remove the finalizer and re-add the module. The module reconciles back to a healthy state and all existing resources are preserved.

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

4. Add the Istio module again. The Istio CR returns to the `Ready` state within a few seconds.

### Remove the Istio Module

If you intentionally deleted the Istio module, you must clean up the blocking resources yourself before the deletion completes.

1. To identify which resources are blocking the deletion, run:

    ```bash
    kubectl logs -n kyma-system -l app=istio-controller-manager --tail=100 | grep "resource is blocking"
    ```

2. Remove the listed resources.
    
Once all blocking resources are removed, the Istio module deletion resumes automatically.