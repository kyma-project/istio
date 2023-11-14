# SAP BTP, Kyma runtime migration

> **NOTE**: This documentation is relevant for SAP BTP, Kyma runtime only and does not apply to open-source Kyma.

## Scenarios

### Note about clusters with existing Istio CR

If there is already Istio CR on the cluster, with name different from `default`, the new Istio CR managed by the Lifecycle Manager will end up in the Error state.
Consider running rename-to-default.sh script to move your custom Istio CR configuration to the new default one. It will also remove the old custom Istio CR during the execution.

### Provisioning of Istio CR using Lifecycle Manager in a new cluster

If there is no Istio custom resource (CR), then Lifecycle Manager provisions the default Istio CR defined in the Istio ModuleTemplate. The migration
adds the Istio module to the Kyma CR.


### Provisioning of Istio CR using Lifecycle Manager in a cluster with existing modules

If there is no Istio CR, then Lifecycle Manager provisions the default Istio CR defined in the Istio ModuleTemplate. The migration
adds the Istio module to the Kyma CR without overwriting existing module configuration.

## Migration test process

### Test scenarios

Apply the ModuleTemplate for both `fast` and `regular` channels to Dev Control Plane.

#### SAP BTP, Kyma runtime clusters without existing modules

1. Create a Dev SAP BTP, Kyma runtime cluster.
2. Execute the migration.
3. Verify that `istio-controller-manager` is installed and the Istio CR's status is `Ready`.

#### SAP BTP, Kyma runtime cluster with an existing module

1. Create a Dev SAP BTP, Kyma runtime cluster.
2. Add the Keda module to the Kyma CR.
   ```yaml
   spec:
     modules:
       - name: keda
   ```
3. Execute the migration.
4. Verify that `istio-controller-manager` is installed and the Istio CR's status is `Ready`.

## Module's rollout and migration

### Preparations

Executing `kcp taskrun` requires the path to the kubeconfig file of the corresponding Gardener project and permissions to list/get shoots.

### Dev

#### Prerequisites

- Reconciliation is disabled for the Dev environment. See PR #4600 in the `kyma/management-plane-config` repository.

#### Migration procedure

1. Apply the ModuleTemplate for both `fast` and `regular` channels to Dev Control Plane.
2. Verify that the ModuleTemplate in the `fast` and `regular` channels is available on SAP BTP, Kyma runtime clusters of the Dev environment.
3. Use `kcp login` to log in to Dev and run the migration script on all SAP BTP, Kyma runtime clusters. To do that, you can use the following command:
   ```shell
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} -t all -- ./managed-kyma-migration.sh
   ```
4. Verify that the migration worked as expected by checking the status of Istio manifests on Control Plane.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio
   ```

### Stage

Perform the rollout to Stage together with the SRE team. Since they have already performed the rollout for other modules, they might suggest a different rollout strategy.

#### Prerequisites

- Reconciliation is disabled for the Stage environment. See PR #4601 to the `kyma/management-plane-config` repository.

#### Migration procedure
1. Push the module to `experimental` channel in `kyma/module-manifests `repository.
2. Test that experimental channel deploys as expected by manually enabling it on a Stage managed cluster
3. Apply the ModuleTemplate for both `fast` and `regular` channels to Stage Control Plane.
4. Verify that the ModuleTemplate in the `fast` and `regular` channels is available in SAP BTP, Kyma runtime clusters of the Stage environment.
5. Use `kcp login` to log in to Stage, select a few SAP BTP, Kyma runtime clusters on `Kyma-Test/Kyma-Integration`, and run `managed-kyma-migration.sh` on them using `kcp taskrun`.
6. Verify if the migration was successful on the SAP BTP, Kyma runtime clusters by checking the status of Istio CR and the reconciler's components.
7. Run `managed-kyma-migration.sh` for all SKRs in `Kyma-Test` and `Kyma-Integration` global accounts.
8. Verify if the migration worked as expected.
9. Run `managed-kyma-migration.sh` for the whole Canary landscape.
10. Verify if the migration worked as expected.
11. If script failed with following log: `More than one Istio CR present on the cluster. Script rename-to-default.sh might be required`, contact the customer to agree on solution. We propose to execute rename-to-default.sh script.
### Prod

Perform the rollout to Prod together with the SRE team. Since they have already performed the rollout for other modules, they might suggest a different rollout strategy.

#### Prerequisites

- Reconciliation is disabled for the Prod environment. See PR #4602 to the `kyma/management-plane-config` repository.

#### Migration procedure

1. Commit the module manifest to the `regular` and `fast` channels in the `kyma/module-manifests` internal repository.
2. Verify that the ModuleTemplates are present in the `kyma/kyma-modules` internal repository.
3. Verify that the ModuleTemplate in both channels are available on `Prod` environment SKRs.
4. Use `kcp login` to log in to Prod, select some SAP BTP, Kyma runtime clusters on `Kyma-Test/Kyma-Integration`, and run `managed-kyma-migration.sh` on them using `kcp taskrun`.
5. Verify if the migration was successful on the SAP BTP, Kyma runtime clusters by checking the status of Istio CR and the reconciler's components.
6. Run `managed-kyma-migration.sh` for all SAP BTP, Kyma runtime clusters in Trial global accounts.
7. Verify if the migration worked as expected.
8. Run `managed-kyma-migration.sh` for the whole Factory landscape.
9. Verify if the migration worked as expected.
10. If script failed with following log: `More than one Istio CR present on the cluster. Script rename-to-default.sh might be required`, contact the customer to agree on solution. We propose to execute rename-to-default.sh script 