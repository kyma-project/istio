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

#### Migration procedure

1. Push module to `experimental` channel in `kyma/module-manifests `repository (PR #162).
2. Push the kustomization change for `experimental` in `kyma/kyma-modules` repository (PR #403)
3. Verify that the ModuleTemplate is present in the experimental channel in the `kyma/kyma-modules` internal repository.
4. Create ModuleTemplate for `fast` and `regular` by adapting the `metadata.name` and `spec.channel` of the `experimental` ModuleTemplate. In this way, we ensure that we also use a ModuleTemplate created by the submission pipeline for the fast and regular channel.
5. Apply the ModuleTemplate for `fast` and `regular` channels to Dev Control Plane.
5. Verify that the ModuleTemplate in the `experimental`, `fast` and `regular` channels is available on SAP BTP, Kyma runtime clusters of the Dev environment.
6. Merge PR (#4624) in `kyma/management-plane-config` responsible for disabling Istio reconciliation and setting Istio as a default module on dev. Depending on the outcome adjust this those actions to one PR for the stage and prod.
7. Get permissions to execute scripts for `kcp taskrun` and use `kcp login` to log in to Dev and run the migration script on all SAP BTP, Kyma runtime clusters. To do that, you can use the following command:
   ```shell
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} -t all -- ./managed-kyma-migration.sh
   ```
8. Verify that the migration worked as expected by checking the status of Istio manifests on Control Plane.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio
   ```
9. Verify that no APIGateway module is in Warning state. This can be done by using the [API-Gateway Monitoring Dashboard](https://plutono.cp.dev.kyma.cloud.sap/d/6meO06VSk/modules-api-gateway?orgId=1).
10. If there are APIGateway CRs in the warning state, the cluster might have been created without Istio Module. In this case it needs to be enabled in the Kyma CR manually.


### Stage

Note: We don't disable Istio reconciliation at the beginning because there would be a time window between that and setting Istio as a default module in which new clusters would not work out of the box (no istio there).
#### Prerequisites

- Reconciliation is disabled for the Stage environment. See PR #4601 to the `kyma/management-plane-config` repository.
- Prepare migration command for Kyma Integration Test Service accounts, make sure to have privileges to execute that.

#### Migration procedure
We skip the new cluster here because we would end up in the error state because there is no Istio, there is api gateway and reconciliation is not disabled yet
explain why we disable reconciliation later
If we would disable Istio reconciliation at the beginning the new clusters created in the time window between making Istio a default module for stage environment would end up not working because of lack of the Istio Module there.
Execute migration script + post tests on testing stage cluster
1. Test already existing clusters if Istio Module is deployed with Istio CR with status Ready. Upgrade, experimental channel, SRE disable reconciliation per this one testing cluster + 2nd script with hardcoded given cluster
2. Execute migration command for Kyma Integration Test Service account to experimental Istio
3. Apply manually the ModuleTemplate for both `fast` and `regular` channels to Stage Control Plane.
4. Verify that the ModuleTemplate in the `fast` and `regular` channels is available in SAP BTP, Kyma runtime clusters of the Stage environment.
5. Disable Istio reconciliation for the Stage environment. See PR #4601 to the `kyma/management-plane-config` repository. If possible depending on the dev migration output try to combine it in one PR with the next step.
6. Merge PR (#4626) in `kyma/management-plane-config` responsible for setting Istio as a default module on stage. + best if we can disable reconciliation together (Depending on what happens on dev)
7. Use `kcp login` to log in to Stage, select a few SAP BTP, Kyma runtime clusters on `Kyma-Test/Kyma-Integration`, and run `managed-kyma-migration.sh` on them using `kcp taskrun`. // run global migration script for default channel
8. Verify if the migration was successful on the SAP BTP, Kyma runtime clusters by checking the status of Istio CR and the reconciler's components. // Check dashboard + smart small taskrun to verify that
9. If script failed with following log: `More than one Istio CR present on the cluster. Script rename-to-default.sh might be required`, contact the customer to agree on solution. We propose to execute rename-to-default.sh script.
10. Migrate Kyma Integration to default channel

### Prod

#### Prerequisites

- Reconciliation is disabled for the Prod environment. See PR #4602 to the `kyma/management-plane-config` repository.

#### Migration procedure
This should be exactly like stage except merging channels, Verify if experimental is available in prod if it's not then we adjust step with testing kyma integration. Then we would use moduletemplateref - it means 3rd script
1. Push the module manifest to the `regular` and `fast` channels in the `kyma/module-manifests` internal repository (PR #163, #164).
2. Push kustomization change for `fast` and `regular` in `kyma/kyma-modules` repository (PR #401)
3. Verify that the ModuleTemplates are present in the `kyma/kyma-modules` internal repository.
4. Verify that the ModuleTemplate in both channels are available on `Prod` environment SKRs.
5. Merge PR (#4627) in `kyma/management-plane-config` responsible for setting Istio as a default module on prod.
6. Use `kcp login` to log in to Prod, select some SAP BTP, Kyma runtime clusters on `Kyma-Test/Kyma-Integration`, and run `managed-kyma-migration.sh` on them using `kcp taskrun`.
7. Verify if the migration was successful on the SAP BTP, Kyma runtime clusters by checking the status of Istio CR and the reconciler's components.
8. Run `managed-kyma-migration.sh` for all SAP BTP, Kyma runtime clusters in Trial global accounts.
9. Verify if the migration worked as expected.
10. Run `managed-kyma-migration.sh` for the whole Factory landscape.
11. Verify if the migration worked as expected.
12. If script failed with following log: `More than one Istio CR present on the cluster. Script rename-to-default.sh might be required`, contact the customer to agree on solution. We propose to execute rename-to-default.sh script.