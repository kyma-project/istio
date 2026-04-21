# Support for Gateway API CRD Installation

## Status
<!--- Specify the current state of the ADR, such as whether it is proposed, accepted, rejected, deprecated, superseded, etc. -->
Proposed

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->
<!--What is the issue that we're seeing that is motivating this decision or change?-->
The Gateway API is an official Kubernetes project that provides a collection of APIs for L4 and L7 traffic routing and management. It represents the next generation of Kubernetes Ingress, Load Balancing, and Service Mesh APIs, designed from the outset to be generic, expressive, and role-oriented.

In Istio, to fully utilize Ambient mode's capabilities, especially waypoint proxies, we need to support Gateway API CRD installation in the Istio CR. Supporting this feature in the Istio CR enables managing the lifecycle of Gateway API CRDs and eliminates the need for manual user installation.

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
<!--What is the change that we're proposing and/or doing?-->

We will add a new feature wrapper for settings regarding Gateway API support in the Istio CR. The initial field of this feature setting will be responsible for Gateway API CRD installation and uninstallation in the Istio CR. 
The feature will be experimental only for now. In the future it will be promoted outside the experimental. 
Istio controller will manage the lifecycle of the Gateway API CRD, which are properly labeled and hence managed by the Istio module. The module never installs or removes the Gateway API
CRDs unless explicitly instructed to do so through this field.


### Istio Custom Resource Configuration
1. Add the new **gatewayAPI** struct:
   - Location: **experimental** struct in the Istio CR specification
   - Type: pointer (optional, omitempty - absent from the object when unset)


1. Add the field **enableCRD**:
    - Location: **gatewayAPI** struct in the Istio CR specification 
    - Type: `*bool` (optional, omitempty - absent from the object when unset)
    - Default: none (when unset disabled)
    - UI Integration: For now, not configurable and not displayed in Kyma dashboard as it is an experimental feature only. When promoted outside experimental, it will be configurable and displayed in Kyma dashboard.
    - Metrics added to track the usage of the feature.
2. Validation:
    - Boolean validation via Istio CRD

Sample Istio CR with the feature enabled:
```yaml
...
spec:
  experimental:
    gatewayAPI:
      enableCRD: true
```

This gives space for future expansion of the **gatewayAPI** struct with more settings related to Gateway API support, such as configuration of specific versions, release channels, or other related features.

### Controller Logic

#### Ownership Model

The module uses the label `kyma-project.io/module=istio` to mark CRDs it owns.
**Only labeled CRDs are ever modified or deleted by the module.** This is the central
invariant of the entire lifecycle:

- CRDs created by the module are always labeled at creation time.
- CRDs found on the cluster without this label are treated as user-managed and are
  never touched — neither updated nor deleted.
- The user can hand over ownership of an existing CRD to the module by manually adding
  the label. From the next reconciliation cycle onward, the module will manage it.

#### Installation and Reconciliation (`gatewayAPI.enableCRD: true`)

On every reconciliation cycle, before proceeding with the Istio installation, the
installer processes each Gateway API CRD from the embedded bundle and takes one of the following actions:

| CRD state on cluster       | Action |
|----------------------------------------|---|
| Not present                            | Created and labeled as module-owned |
| Present, module-owned, version matches | No change (idempotent) |
| Present, module-owned, version differs | Updated to bundle version |
| Present, **not** module-owned          | Skipped – label hint logged |

> [!NOTE] 
> When some CRDs already exist in the cluster without the module label
> (for example, installed manually by the user or by another tool), the module logs a warning
> listing each unmanaged CRD and the exact `kubectl label` action required to transfer
> ownership. Istio installation continues regardless — unmanaged CRDs do not block the
> reconciliation.

> [!IMPORTANT]  
> A partial state is possible — the module can manage some CRDs while leaving others unmanaged. 
> The module handles each CRD independently, so partial ownership is a valid and
> stable runtime state. The user is responsible for deciding whether to transfer
> ownership of unmanaged CRDs.

### Uninstallation – Feature Disabled (`gatewayAPI.enableCRD: false` after previously `true`)

When **gatewayAPI.enableCRD** is explicitly set to `false` (or removed) after having been
previously enabled, the module uninstalls Gateway API CRDs as part of the
next reconciliation cycle. The cleanup checks:

1. Only CRDs with the module ownership label are candidates for deletion.
2. Before deleting any CRD, the module scans the cluster for existing Gateway API
   custom resources that are instances of CRDs managed by the Istio module.
3. If **any** such resources are found, deletion is **blocked**. The module sets the
   proper condition on the Istio CR, logs each blocking resource by
   name and namespace, and returns an error. The reconciler retries deletion in the next cycle.
4. The module deletes the labeled CRDs only when the cluster doesn't contain Gateway API custom resources whose parent CRDs are managed.

> [!NOTE]
> Unmanaged CRDs (no module label) are never deleted, even when the feature
> is disabled. They remain on the cluster as the user left them.

> [!NOTE]
> The cleanup path is only triggered when the feature was previously
> explicitly enabled — tracked via the last-applied configuration annotation. Setting
> `gatewayAPI.enableCRD: false` in a new Istio CR that never had it enabled is a
> complete no-op. This prevents accidental cleanup.

### Uninstallation – Istio CR Deleted

When the Istio CR itself is deleted, the same uninstall Gateway API CRDs logic
applies: only module-labeled CRDs are removed, and only if no corresponding Gateway API custom
resources are present in the cluster. If blocking resources exist, the
proper condition is set, and the finalizer is **not** removed, keeping
the Istio CR in a terminating state until the user cleans up the Gateway API resources.

## Consequences
<!--- Discuss the impact of this change, including what becomes easier or more complicated as a result. -->
<!--What becomes easier or more difficult to do because of this change?-->

### Benefits

- **No silent side effects.** The module never touches resources it does not own. Users
  who manage Gateway API CRDs independently are not affected.
- **Partial installation is stable.** The module does not require all-or-nothing
  ownership. Mixed states (some CRDs owned, some not) are handled gracefully on every
  reconciliation.
- **Version is pinned to the bundle.** The module does not attempt to detect or
  reconcile arbitrary versions already in the cluster. It only applies the version
  embedded in the controller binary. Version upgrades happen automatically on the next
  reconciliation after a controller update.
- **Improved user experience.** Gateway API CRD lifecycle is fully automated through the Istio controller, removing the need for manual installation and enabling users to leverage the full potential of Ambient mode and waypoint proxies as also Gateway API CRs.


### Trade-Offs
- **Ownership transfer is manual and opt-in.** There is no automatic adoption of
  pre-existing CRDs. The user must explicitly add the module label to signal intent.
- **Gateway API version maintenance.** The module pins Gateway API CRDs to the version embedded in the controller binary. Compatibility with Istio versions must be tracked and communicated, and potential breaking changes must be planned during Istio upgrades.
- **Increased controller complexity.** Managing the full CRD lifecycle — installation, updates, ownership tracking, and safe deletion — adds non-trivial logic to the controller reconciliation flow.
- **Once managed by module no easy way back.** Once user gives the ownership of a CRD to the module, it won't be an easy way back to manage it by user. This means that even if the user later decides to manage the CRD independently, they must first remove the module setting - manually delete CRs, module will delete CRDs.