# Managed Kyma migration
> **NOTE**: This documentation is relevant for managed Kyma only and does not apply to OS Kyma.

## Prerequisites
- The Kyma reconciler will stop the reconciliation of the Istio module when the `istios.operator.kyma-project.io` is available in version `v1alpha2` on the SKR, since
  version `v1alpha1` is already available on the SKRs. The `v1alpha2` configuration was added with this [PR](https://github.com/kyma-project/control-plane/pull/2847).

## Scenarios

### Istio CR created by the customer exists on SKR
If the Istio CR is already present, avoid overwriting existing configuration. Do not create 
the default Istio CR, but use the existing CR. The migration adds the Istio module to the Kyma CR with `customResourcePolicy` set to `Ignore`.

### Provisioning of Istio CR via Lifecycle-Manager in a new cluster
If there is no Istio CR, then Lifecycle Manager provisions the default Istio CR defined in the Istio module. The migration
adds the Istio module to the Kyma CR.

### Provisioning of Istio CR via Lifecycle Manager in a cluster with existing modules
If there is no Istio CR, then Lifecycle Manager provisions the default Istio CR defined in the Istio module. The migration
adds the Istio module to the Kyma CR without overwriting existing module configuration.

## Migration test process

### Test scenarios
1. Set `metadata.name` in the module template to `istio-regular`.
2. Set `spec.channel` in the module template to `regular`.
3. Apply the module template to DEV Control Plane.

#### SKR with existing Istio CR
1. Create Dev SKR.
2. Create Istio CR on SKR.
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
3. Execute the migration.
4. Verify that `istio-manager` is installed and the Istio CR's status is `Ready`.
5. Verify that the Kyma reconciler does not reconcile the Istio component anymore.
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```

#### SKR without existing Istio CR
1. Create Dev SKR.
2. Execute the migration.
3. Verify that `istio-manager` is installed and the Istio CR's status is `Ready`.
4. Verify that the Kyma reconciler does not reconcile the Istio component anymore.
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```

#### SKR with an existing module
1. Create Dev SKR.
2. Add the Keda module to the Kyma CR.
   ```yaml
   spec:
     modules:
     - name: keda
   ```
3. Execute the migration.
4. Verify that `istio-manager` is installed and the Istio CR's status is `Ready`.
5. Verify that the Kyma reconciler does not reconcile the Istio component anymore.
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```   

## Module's rollout and migration

### Preparations
1. Executing `kcp taskrun` requires the path to the kubeconfig file of the corresponding Gardener project and permissions to list/get shoots.
2. Set `metadata.name` in the module template to `istio-regular`.
3. Set `spec.channel` in the module template to `regular`.

### Dev
1. Apply the module template to Dev Control Plane.
2. `kcp login` to Dev and run the migration script on all SKRs. To do that, you can use the following command:
   ```shell 
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} --gardener-namespace kyma-dev -t all -- ./migrate.sh
   ```
3. Verify that the migration worked as expected by checking the status of Istio manifests on Control Plane.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio
   ```

### Stage

Perform the rollout to Stage together with the SRE team. Since they have already performed the rollout for other modules, they might suggest a different rollout strategy.

1. Apply the module template to Stage Control Plane.
2. `kcp login` to Stage, select some SKRs on `Kyma-Test/Kyma-Integration`, and run `migrate.sh` on them using `kcp taskrun`.
3. Verify if the migration was successful on the SKRs by checking the status of Istio CR and the reconciler's components.
4. Run `migrate.sh` for all SKRs in Kyma-Test and Kyma-Integration global accounts.
5. Verify if the migration worked as expected.
6. Run `migrate.sh` for the whole Canary landscape.
7. Verify if the migration worked as expected.