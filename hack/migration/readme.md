# Migration documentation

## Prerequisites
- The Kyma reconciler will stop the reconciliation of the Istio module when the `istios.operator.kyma-project.io` is available in version `v1alpha2` on the SKR, since
  version `v1alpha1` is already available on the SKRs. This configuration is added in this [PR](https://github.com/kyma-project/control-plane/pull/2847).

## Scenarios

### Customer created Istio CR exists on the SKR
If the Istio CR is already present, we want to avoid overwriting existing configuration. Therefore, we will not create 
the default Istio CR, but use the existing CR.

### Provisioning of Istio CR via Lifecycle-Manager
If there is no Istio CR, then Lifecycle-Manager will provision the default Istio CR defined in the Istio module.

## Migration test process

### Preparations
1. Executing `kcp taskrun` requires the path to the kubeconfig file of the corresponding Gardener project and permissions to list/get shoots

### Test scenarios
1. Set the `spec.channel` in the module template to `regular`
2. Apply module template to DEV Control Plane
3. Execute migration script on a DEV SKR cluster with existing Istio CR  
   ```shell 
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} --gardener-namespace kyma-dev -t shoot={SHOOT NAME} -- ./migrate.sh
   ```
4. Execute migration script on a DEV SKR cluster without existing Istio CR  
   ```shell 
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} --gardener-namespace kyma-dev -t shoot={SHOOT NAME} -- ./migrate.sh
   ```
5. Verify that istio-manager is installed and reconciliation is executed
6. Verify that the Kyma reconciler does not reconcile the Istio module anymore
   ```shell
   # List reconciliations to get the last scheduling ID
   kcp rc -c {SHOOT}
   # List reconciled components
   kcp rc info -i {SCHEDULING ID}
   ```

## Initial module rollout and migration

### Preparations
1. Executing `kcp taskrun` requires the path to the kubeconfig file of the corresponding Gardener project and permissions to list/get shoots
2. Set the `spec.channel` in the module template to `regular`
3. Create a Draft PR to the internal kyma-modules repository with the module template. This PR will be merged later for the rollout in production.

### Rollout on Dev
1. Apply the module template to Dev Control Plane
2. `kcp login` to Dev and run migration script on all SKRs with a parallelism of 8  
   ```shell 
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} --gardener-namespace kyma-dev -t all -p 8 -- ./migrate.sh
   ```
3. Verify migration worked as expected by checking Istio manifests status on Control Plane
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio
   ```

### Rollout on Stage

Rollout to Stage should be done together with the SRE team.

1. Apply the module template to Stage Control Plane
2. `kcp login` to Stage, select some SKRs on Kyma-Test/Kyma-Integration and run `migrate.sh` on them using `kcp taskrun`
   > **Note**: In this documentation shoots are used for targeting, but also subaccounts may be used.
3. Verify migration worked as expected 
4. Run `migrate.sh` for all SKRs in Kyma-Test and Kyma-Integration global accounts
5. Verify migration worked as expected
6. Run `migrate.sh` for whole Canary landscape
7. Verify migration worked as expected