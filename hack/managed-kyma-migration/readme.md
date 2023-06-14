# Managed Kyma migration documentation
> **NOTE**: This documentation is for managed Kyma only and is not applicable for OS Kyma.

## Prerequisites
- The Kyma reconciler will stop the reconciliation of the Istio module when the `istios.operator.kyma-project.io` is available in version `v1alpha2` on the SKR, since
  version `v1alpha1` is already available on the SKRs. This configuration is added in this [PR](https://github.com/kyma-project/control-plane/pull/2847).

## Scenarios

### Customer created Istio CR exists on the SKR
If the Istio CR is already present, we want to avoid overwriting existing configuration. Therefore, we will not create 
the default Istio CR, but use the existing CR. The migration will add the Istio module to the Kyma CR with the `customResourcePolicy` set to `Ignore`.

### Provisioning of Istio CR via Lifecycle-Manager in new cluster
If there is no Istio CR, then Lifecycle-Manager will provision the default Istio CR defined in the Istio module. The migration
will add the Istio module to the Kyma CR.

### Provisioning of Istio CR via Lifecycle-Manager in cluster with existing modules
If there is no Istio CR, then Lifecycle-Manager will provision the default Istio CR defined in the Istio module. The migration
will add the Istio module to the Kyma CR without overwriting existing module configuration.

## Migration test process

### Test scenarios
1. Set `metadata.name` in the module template to `istio-regular`
2. Set the `spec.channel` in the module template to `regular`
3. Apply module template to the DEV Control Plane

**SKR with existing Istio CR**
1. Create Dev SKR
2. Create Istio CR ony SKR
   ```yaml 
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: Istio
   metadata:
     name: default
     namespace: kyma-system 
     labels:
       app.kubernetes.io/name: default
   spec:
     config:
       numTrustedProxies: 1
   ```
3. Execute migration
4. Verify that istio-manager is installed and Istio CR status is Ready
5. Verify that the Kyma reconciler does not reconcile the Istio component anymore
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```

**SKR without existing Istio CR**   
1. Create Dev SKR
2. Execute migration
3. Verify that istio-manager is installed and Istio CR status is Ready
4. Verify that the Kyma reconciler does not reconcile the Istio component anymore
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```

**SKR with existing module**
1. Create Dev SKR
2. Add module keda to Kyma CR
   ```yaml
   spec:
     modules:
     - name: keda
   ```
3. Execute migration
4. Verify that istio-manager is installed and Istio CR status is Ready
5. Verify that the Kyma reconciler does not reconcile the Istio component anymore
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```   

## Module rollout and migration

### Preparations
1. Executing `kcp taskrun` requires the path to the kubeconfig file of the corresponding Gardener project and permissions to list/get shoots
2. Set `metadata.name` in the module template to `istio-regular`
3. Set `spec.channel` in the module template to `regular`

### Dev
1. Apply the module template to the Dev Control Plane
2. `kcp login` to Dev and run migration script on all SKRs, the following command might be used:
   ```shell 
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} --gardener-namespace kyma-dev -t all -- ./migrate.sh
   ```
3. Verify migration worked as expected by checking Istio manifests status on Control Plane
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio
   ```

### Stage

Rollout to Stage should be done together with the SRE team. Since they have already done the rollout for other modules, they might suggest a different rollout strategy.

1. Apply the module template to Stage Control Plane
2. `kcp login` to Stage, select some SKRs on Kyma-Test/Kyma-Integration and run `migrate.sh` on them using `kcp taskrun`
3. Verify migration worked on the SKRs by checking Istio CR status and reconciler components
4. Run `migrate.sh` for all SKRs in Kyma-Test and Kyma-Integration global accounts
5. Verify migration worked as expected
6. Run `migrate.sh` for whole Canary landscape
7. Verify migration worked as expected