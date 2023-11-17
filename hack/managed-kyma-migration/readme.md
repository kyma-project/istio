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
6. Verify that the ModuleTemplate in the `experimental`, `fast` and `regular` channels is available on SAP BTP, Kyma runtime clusters of the Dev environment.
7. Merge PR (#4624) in `kyma/management-plane-config` responsible for disabling Istio reconciliation and setting Istio as a default module on dev. Depending on the outcome adjust this those actions to one PR for the stage and prod.
8. Get permissions to execute scripts for `kcp taskrun` and use `kcp login` to log in to Dev and run the migration script on all SAP BTP, Kyma runtime clusters. To do that, you can use the following command:
   ```shell
   kcp taskrun --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} -t all -- ./managed-kyma-migration.sh
   ```
9. Verify that the migration worked as expected by checking the status of Istio manifests on Control Plane.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio
   ```
10. Verify that no APIGateway module is in Warning state. This can be done by using the [API-Gateway Monitoring Dashboard](https://plutono.cp.dev.kyma.cloud.sap/d/6meO06VSk/modules-api-gateway?orgId=1).
11. If there are APIGateway CRs in the warning state, the cluster might have been created without Istio Module. In this case it needs to be enabled in the Kyma CR manually.


### Stage

We don't want to disable Istio reconciliation at the beginning of the rollout, because there would be a big time window between stopping the istio reconciliation and setting Istio as a default module.
During this time new clusters can not be created successfully, since Istio as a dependency for API-Gateway module is missing.

#### Migration procedure
##### Test migration
1. Execute migration script `migration-testing/migrate-local-moduletemplate.sh` that uses local module template for fast channel to migrate Service Account provided for Upgrade testing. At this point we want to skip the installation test on a new cluster, 
because Istio module is not a default module at this time and Istio component is still installed by the reconciler.
2. Get privileges(CAM-profile) to execute migration script using taskrun for kyma-integration Global Account
3. Trigger SRE to disable reconciliation for kyma-integration Global Account. This is necessary, because we didn't disable istio module reconciliation, yet.
4. Execute migration script `migration-testing/migrate-local-moduletemplate.sh` that uses local module template for fast channel to migrate kyma-integration Global Account
   ```shell
   kcp taskrun -p 16 --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} -t account="$GLOBAL_ACCOUNT_KYMA_INTEGRATION" -- ./migrate-local-moduletemplate.sh 
   ```
5. Verify that the number of Istio manifests on Stage Control Plane is equal or higher (because of test clusters from step 1) to the number of runtimes on Stage for kyma-integration Global Account. First returns the number of runtimes, second the number of manifests.
   ```shell
   kcp rt -o json | jq '.totalCount'
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v "NAMESPACE" | wc -l
   ```
6. Verify test migration of kyma-integration Global Account was successful by checking the status of Istio manifests on Stage Control Plane. The following script will print out istio manifests that are not Ready.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v Ready
   ```

##### Migration rollout (done by SRE)
1. Apply manually the ModuleTemplate for both `fast` and `regular` channels to Stage Control Plane.
2. Verify that the ModuleTemplate in the `fast` and `regular` channels is available in SAP BTP, Kyma runtime clusters of the Stage environment.
3. Merge PR (#4624) in `kyma/management-plane-config` responsible for disabling Istio reconciliation and setting Istio as a default module
   TODO: Update PR to contain disabling istio and making it default module. This is dependent on the outcome of the dev migration.
4. Execute migration script `migrate.sh` for default channel migration on all clusters
5. If script failed with following log: `More than one Istio CR present on the cluster. Script rename-to-default.sh might be required`, contact the customer to agree on solution. We propose to execute rename-to-default.sh script.

##### Verify migration
1. Wait for SRE to finish the migration for all clusters
2. Check the [Istio Module](https://plutono.cp.stage.kyma.cloud.sap/d/hTm72lVIz/modules-istio?orgId=1) and [API Gateway Module](https://plutono.cp.stage.kyma.cloud.sap/d/6meO06VSk/modules-api-gateway?orgId=1) status on the dashboards
3. Verify that the number of Istio manifests on Stage Control Plane is equal to the number of runtimes on Stage. First returns the number of runtimes, second the number of manifests.
   ```shell
   kcp rt --account "$GLOBAL_ACCOUNT_KYMA_INTEGRATION" -o json | jq '.totalCount'
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v "NAMESPACE" | wc -l
   ```
4. Verify the migration was successful by checking the status of Istio manifests on Stage Control Plane. The following script will print out istio manifests that are not Ready.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v Ready
   ```
5. Migrate istio module in kyma-integration Global Account to default channel and remove remote module template used for migration testing
  TODO: Add script
6. Trigger SRE to enable reconciliation for kyma-integration Global Account.


### Prod

##### Test migration
1. Execute migration script `migration-testing/migrate-local-moduletemplate.sh` that uses local module template for fast channel to migrate Service Account provided for Upgrade testing. At this point we want to skip the installation test on a new cluster,
   because Istio module is not a default module at this time and Istio component is still installed by the reconciler.
2. Get privileges (CAM-profile) to execute migration script using taskrun for kyma-integration Global Account
3. Trigger SRE to disable reconciliation for kyma-integration Global Account. This is necessary, because we didn't disable istio module reconciliation, yet.
4. Execute migration script `migration-testing/migrate-local-moduletemplate.sh` that uses local module template for fast channel to migrate kyma-integration Global Account
      ```shell
   kcp taskrun -p 16 --gardener-kubeconfig {PATH TO GARDENER PROJECT KUBECONFIG} -t account="$GLOBAL_ACCOUNT_KYMA_INTEGRATION" -- ./migrate-local-moduletemplate.sh 
   ```
5. Verify that the number of Istio manifests on Prod Control Plane is equal or higher (because of test clusters from step 1) to the number of runtimes on Prod for kyma-integration Global Account. First returns the number of runtimes, second the number of manifests.
   ```shell
   kcp rt --account "$GLOBAL_ACCOUNT_KYMA_INTEGRATION" -o json | jq '.totalCount'
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v "NAMESPACE" | wc -l
   ```
6. Verify test migration of kyma-integration Global Account was successful by checking the status of Istio manifests on Prod Control Plane. The following script will print out istio manifests that are not Ready.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v Ready
   ```
7. Push the module manifest to the `regular` and `fast` channels in the `kyma/module-manifests` internal repository (PR #163, #164).
8. Push kustomization change for `fast` and `regular` in `kyma/kyma-modules` repository (PR #401)
9. Verify that the ModuleTemplates are present in the `kyma/kyma-modules` internal repository.
10. Verify that the ModuleTemplate in both channels are available on `Prod` environment SKRs.

##### Migration rollout (done by SRE)
1. Merge PR (#4627) in `kyma/management-plane-config` responsible for disabling Istio reconciliation and setting Istio as a default module
   TODO: Update PR to contain disabling istio and making it default module. This is dependent on the outcome of the dev migration.
2. Execute migration script `migrate.sh` for default channel migration on all clusters
3. If script failed with following log: `More than one Istio CR present on the cluster. Script rename-to-default.sh might be required`, contact the customer to agree on solution. We propose to execute rename-to-default.sh script.

##### Verify migration
1. Wait for SRE to finish the migration for all clusters
2. Check the [Istio Module](https://plutono.cp.kyma.cloud.sap/d/hTm72lVIz/modules-istio?orgId=1) and [API Gateway Module](https://plutono.cp.kyma.cloud.sap/d/6meO06VSk/modules-api-gateway?orgId=1) status on the dashboards
3. Verify that the number of Istio manifests on Prod Control Plane is equal to the number of runtimes on Prod. First returns the number of runtimes, second the number of manifests.
   ```shell
   kcp rt -o json | jq '.totalCount'
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v "NAMESPACE" | wc -l
   ```
4. Verify the migration was successful by checking the status of Istio manifests on Prod Control Plane. The following script will print out istio manifests that are not Ready.
   ```shell
   kubectl get manifests -n kcp-system -o custom-columns=NAME:metadata.name,STATE:status.state | grep istio | grep -v Ready
   ```
5. Migrate istio module in kyma-integration Global Account to default channel and remove remote module template used for migration testing
   TODO: Add script
6. Trigger SRE to enable reconciliation for kyma-integration Global Account.

##### Clean up
1. Remove the experimental ModuleTemplate.
TODO: Clarify how the process will look like.

