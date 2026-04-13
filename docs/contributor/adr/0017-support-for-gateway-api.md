# <!--- Provide title -->

## Status
<!--- Specify the current state of the ADR, such as whether it is proposed, accepted, rejected, deprecated, superseded, etc. -->
Proposed

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->
<!--What is the issue that we're seeing that is motivating this decision or change?-->
The Gateway API is an official Kubernetes project that provides a collection of APIs for L4 and L7 traffic routing and management. It represents the next generation of Kubernetes Ingress, Load Balancing, and Service Mesh APIs, designed from the outset to be generic, expressive, and role-oriented.

In Istio to fully utilize the capabilities of Ambient mode, especially waypoint proxies, we need to support Gateway API CRD installation in Istio CR. Installation via feature enablement in Istio CR allows management of the lifecycle of Gateway API CRDs and manual installation by user is not needed.

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
<!--What is the change that we're proposing and/or doing?-->

We will add support for Gateway API CRD installation and uninstallation in Istio CR. Feature will be experimental only. Istio controller will manage the lifecycle of the Gateway API CRD, which are properly labeled. 

### Istio CR configuration:
1. Add the **enableGatewayAPI** field: 
    - Location: **Experimental** struct in the Istio CR specification 
    - Type: `*bool` (optional)
    - Default: none (when unset disabled)
    - UI Integration: Not configurable and not displayed in Kyma dashboard as for now it is experimental feature only 
2. Validation:
    - Boolean validation via Istio CRD

### Controller logic:
1. **Installation:** When **enableGatewayAPI** transitions to `true`, check for existing Gateway API CRDs. If found, and they were not applied by Istio controller (check for Istio module label), log and warn user that there already exist Gateway API CRD on cluster. 
2. Gateway API CRD lifecycle management in Istio controller:
    - **Labeling strategy**: Add management labels to distinguish Istio-managed Gateway API CRDs, for example label `kyma-project.io/module=istio`
    - **Deletion protection**: Add warning to Istio CR status and log it in istio controller manager if active CRs of Gateway API exist and **enableGatewayAPI** would be set to false. Add to Istio finalizers to prevent accidental deletion of Gateway API CRDs with active Gateway API CRs on cluster. 
    - **Reconciliation**: Reconcile Gateway API CRD presence and version during Istio CR reconciliation. If **enableGatewayAPI** is `true` but CRD is missing, attempt to reinstall it. If CRD version is incompatible with current Istio version, log warning and attempt to update to supported version if possible.

## Consequences
<!--- Discuss the impact of this change, including what becomes easier or more complicated as a result. -->
<!--What becomes easier or more difficult to do because of this change?-->

### Benefits:
- Ambient mode fully leverages Gateway API capabilities, especially for Waypoint proxies.
- Improved user experience with automated Gateway API CRD installation management through the Istio controller.
- Gateway API enables customers to create additional Istio Ingress Gateways.

### Trade-offs:
- Maintenance of Gateway API versioning and compatibility with Istio versions. Istio documentation does not specify which Gateway API CRD versions are supported, requiring us to track and update them as needed. During Istio upgrades, we must plan for and communicate potential breaking changes to users.
- Potential user confusion if existing Gateway API CRDs are present when enabling the feature.
- Additional complexity in controller logic for managing CRD lifecycle.