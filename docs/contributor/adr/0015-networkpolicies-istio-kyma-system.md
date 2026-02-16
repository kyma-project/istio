# NetworkPolicies for istio-system and kyma-system

## Status
Proposed

## Context
Clusters that enforce a default-deny NetworkPolicy require explicit allow rules for Istio components to install and operate.
This ADR captures those requirements so they can be consistently applied and maintained.

To ensure that Istio components can function properly under default-deny policies,
we need to define the necessary NetworkPolicies for both `istio-system` and `kyma-system` namespaces.
This includes allowing essential egress traffic for DNS and API server access,
as well as ingress and egress rules for control-plane communication between Istio components.

## Decision
Extend Istio Custom Resource (CR) to include a flag for enabling NetworkPolicy support.

When user enables NetworkPolicy support in the Istio CR, apply NetworkPolicies to `istio-system` and `kyma-system` namespaces that allow the following traffic:

1. Allow DNS egress for all module-related pods in both namespaces to the cluster DNS service (TCP/UDP 53).
2. Allow API server access (TCP 443) for:
   - `istio-controller-manager` in `kyma-system`
   - `istiod` (`istio: pilot`) in `istio-system`
   - `istio-cni-node` in `istio-system`
3. Allow control-plane communication between `istiod` and the `istio-ingressgateway`:
   - Allow egress from `istio-ingressgateway` to `istiod` on port 15012 (TCP, XDS protocol).
   - Allow ingress to `istiod` on port 15012 from sidecars and `istio-ingressgateway`.
4. Allow external ingress to `istio-ingressgateway` on TCP 8080/8443 for traffic entering the cluster.
5. Allow ingress to the `istiod` webhook endpoint on TCP 15017 from the API server for validating and mutating operations.
6. Allow `istiod` egress to common JWKS endpoint ports (TCP 80/443) for external JWT verification.
> [!NOTE]
> Allowing traffic to the 443 and 80 is not necessarily sufficient for all cases, as some JWKS endpoints might be running on non-standard ports.
> If users have specific requirements for accessing JWKS endpoints on non-standard ports, it might be required to either allow users to create custom NetworkPolicies in
> the `istio-system` namespace or to provide a way to specify additional allowed ports for `istiod` egress in the Istio CR.
7. Allow user-enabled egress from `istio-ingressgateway` to backend services by permitting egress to specifically labeled pods in user namespaces.

To ensure that the policies are enforced as soon as user enables the setting the Istio module components will be restarted (rollout restart)
to terminate already established TCP connections.

## Consequences

### Extend Istio Custom Resource with NetworkPolicy Support Flag

The Istio Custom Resource Definition will be extended to include a new boolean field, `enableModuleNetworkPolicies`,
which allows users to enable or disable NetworkPolicy support for the Istio components. When this flag is set to `true`,
the necessary NetworkPolicies will be applied to the `istio-system` and `kyma-system` namespaces.
By default, this flag will be set to `false` to avoid applying NetworkPolicies in clusters that do not enforce them,
ensuring backward compatibility and preventing unintended disruptions in such environments.

The field will be placed under the `spec` section of the Istio CR, and will be documented to explain its purpose and the implications of enabling it.
Example:

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
   name: default
   namespace: kyma-system
spec:
   enableModuleNetworkPolicies: true
```

### Create NetworkPolicies for Istio Components

On enabling the `enableModuleNetworkPolicies` flag, the following NetworkPolicies will be created to allow the necessary traffic for Istio components.

1. Allow APIServer and DNS access for `istio-controller-manager` in `kyma-system`:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--allow-istio-controller-manager
  namespace: kyma-system
spec:
  podSelector:
    matchLabels:
      kyma-project.io/module: istio
      app.kubernetes.io/name: istio-operator
  policyTypes:
  - Egress
  egress:
  - ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  - ports:
    - protocol: TCP
      port: 443
```

2. Allow the following for `istiod`:
- Allow egress access to DNS and APIServer.
- Allow ingress access from sidecars and `istio-ingressgateway` on port 15012 for control-plane communication.
- Allow ingress access to the webhook endpoint on port 15017.

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--istio-pilot
  namespace: istio-system
spec:
  podSelector:
    matchLabels:
      istio: pilot
  policyTypes:
  - Egress
  - Ingress
  egress:
  - ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  - ports:
    - protocol: TCP
      port: 443
  ingress:
  - from:
    - podSelector:
        matchLabels:
          security.istio.io/tlsMode: istio
      namespaceSelector: {}
    - podSelector:
        matchLabels:
          istio: ingressgateway
    ports:
    - protocol: TCP
      port: 15012
  - ports:
    - protocol: TCP
      port: 15017
```

3. Allow the following for `istio-ingressgateway`:
- Allow egress access to `istiod` on port 15012 (XDS).
- Allow egress access to DNS.
- Allow ingress access from external traffic on ports 80/443.

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--istio-ingressgateway
  namespace: istio-system
spec:
  podSelector:
    matchLabels:
      istio: ingressgateway
  policyTypes:
  - Egress
  - Ingress
  egress:
  - to:
    - podSelector:
        matchLabels:
          istio: pilot
    ports:
    - protocol: TCP
      port: 15012
  - ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  ingress:
  - ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8443
```

4. Allow egress access to DNS and APIServer for `istio-cni-node`:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--istio-cni-node
  namespace: istio-system
spec:
  podSelector:
    matchLabels:
      k8s-app: istio-cni-node
  policyTypes:
  - Egress
  egress:
  - ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  - ports:
    - protocol: TCP
      port: 443
```

5. Allow access to external (outside of cluster) JWKS endpoints for JWT verification by `istiod`:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--istio-pilot-jwks
  namespace: istio-system
spec:
  podSelector:
    matchLabels:
      istio: pilot
  policyTypes:
  - Egress
  egress:
  - ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 443
```

6. Allow user-enabled egress traffic from `istio-ingressgateway` to backend services by creating a NetworkPolicy
that allows egress to specifically labeled pods in user namespaces:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--istio-ingressgateway-egress
  namespace: istio-system
spec:
  podSelector:
    matchLabels:
      istio: ingressgateway
  policyTypes:
  - Egress
  egress:
  - to:
    - podSelector:
        matchLabels:
          networking.kyma-project.io/from-ingressgateway: allowed
      namespaceSelector: {}
```

### Restart Istio Components to Enforce Policies
To ensure that the newly applied NetworkPolicies take effect immediately, all Istio components will be restarted (rollout restart)
when the user enables the NetworkPolicy support flag in the Istio CR.
This will terminate any existing TCP connections and allow the new policies to be enforced without delay.
This means that a restart should happen whenever the value of `enableModuleNetworkPolicies` is changed from `false` to `true` or vice versa,
to ensure that the correct policies are applied based on the user's choice.

### User impact

To ensure that the connectivity between istio-ingressgateway and user workloads is maintained when NetworkPolicy support is enabled,
users will need to label the pods in their namespaces that should be accessible from the `istio-ingressgateway`
with the label `networking.kyma-project.io/from-ingressgateway: allowed`.

### Default-deny policies in user namespaces
If the user desires to run under a default-deny policy in their namespaces,
they will need to create appropriate NetworkPolicies to allow traffic from the `istio-ingressgateway` to their workloads. 
This will not be automatically handled by the Istio module,
as it is outside the `istio-system` or `kyma-system` namespace, but it will be documented as a requirement for users who want to enable NetworkPolicy support.
