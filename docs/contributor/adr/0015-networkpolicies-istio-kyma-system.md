# NetworkPolicies for istio-system and kyma-system

## Status
Proposed

## Context
This ADR addresses requirements defined in [kyma-project/kyma#18818](https://github.com/kyma-project/kyma/issues/18818).
Clusters that enforce a default-deny NetworkPolicy require explicit allow rules for Istio components to install and operate.
This ADR captures those requirements so they can be consistently applied and maintained.

To ensure that Istio components function properly under default-deny policies,
we must define the necessary NetworkPolicies for both `istio-system` and `kyma-system` namespaces.
This includes allowing essential egress traffic for DNS and API server access,
as well as ingress and egress rules for control-plane communication between Istio components.

## Decision
Extend Istio Custom Resource (CR) to include a flag for enabling NetworkPolicy support.

When the user enables NetworkPolicy support in the Istio CR, apply NetworkPolicies to the `istio-system` and `kyma-system` namespaces that allow the following traffic:

- Allow DNS egress from all module-related Pods in both namespaces to the cluster DNS service (TCP/UDP `53`).
- Allow API server access (TCP `443`) for the following components:
   - `istio-controller-manager` in `kyma-system`
   - `istiod` (`istio: pilot`) in `istio-system`
   - `istio-cni-node` in `istio-system`
- Allow control-plane communication between `istiod` and `istio-ingressgateway`:
   - Allow egress from `istio-ingressgateway` to `istiod` on port `15012` (TCP, XDS protocol).
   - Allow ingress from sidecars and `istio-ingressgateway` to `istiod` on port `15012`.
- Allow external ingress to `istio-ingressgateway` on TCP `8080`/`8443` for traffic entering the cluster.
- Allow ingress from the API server to the `istiod` webhook endpoint on TCP `15017` for validating and mutating operations.
- Allow `istiod` egress to common JWKS endpoint ports (TCP `80`/`443`) for external JWT verification.
  > [!NOTE]
  > Allowing traffic to the 443 and 80 is not necessarily sufficient for all cases, as some JWKS endpoints might be running on non-standard ports.
  > If users have specific requirements for accessing JWKS endpoints on non-standard ports, it might be required to either allow users to create custom NetworkPolicies in
  > the `istio-system` namespace or to provide a way to specify additional allowed ports for `istiod` egress in the Istio CR.
- Allow user-enabled egress traffic from `istio-ingressgateway` to backend services by permitting egress to specifically labeled Pods in user namespaces.
- Allow the following according to https://istio.io/latest/docs/ops/deployment/application-requirements:
  - Allow ingress to `istiod` on TCP `15014` (Control plane monitoring).
  - Allow ingress to `istio-ingressgateway` on TCP `15008` (HBONE mTLS tunnel port, Ambient mode).
  - Allow ingress to `istio-ingressgateway` on TCP `15020` (Merged Prometheus metrics port).
  - Allow ingress to `istio-ingressgateway` on TCP `15021` (Health checks).
  - Allow ingress to `istio-ingressgateway` on TCP `15090` (Envoy Prometheus telemetry).

To ensure that the policies are enforced as soon as the user enables the setting, the Istio module's components must be restarted (rollout restart)
to terminate already established TCP connections.

To increase visibility and user awareness of the applied NetworkPolicies,
the applied resources should be labeled with `kyma-project.io/module: istio` and `kyma-project.io/managed-by: kyma`
to indicate that they are managed by the Istio module and should not be modified by users.

## Consequences

### Istio Custom Resource Extension

Extend the Istio Custom Resource Definition to include a new boolean field, **enableModuleNetworkPolicies**,
which allows users to enable or disable NetworkPolicy support for the Istio components. When this flag is set to `true`,
the necessary NetworkPolicies must be applied to the `istio-system` and `kyma-system` namespaces.
By default, this flag is set to `false` to prevent NetworkPolicies from being applied in clusters where it is not enforced.
This ensures backward compatibility and prevents unintended disruptions in such environments.

Include the field under the **spec** section of the Istio CR, and document its purpose and the implications of enabling it.

See the following example:

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
   name: default
   namespace: kyma-system
spec:
   enableModuleNetworkPolicies: true
```

### NetworkPolicies Applied by Module

| Component                | Namespace    | Port  | Protocol  | Direction | Purpose                                                                                      |
|--------------------------|--------------|-------|-----------|-----------|----------------------------------------------------------------------------------------------|
| `istio-controller-manager` | `kyma-system`  | `53`    | UDP/TCP   | Egress    | DNS resolution                                                                               |
| `istio-controller-manager` | `kyma-system`  | `443`   | TCP       | Egress    | Kubernetes API server access                                                                 |
| `istiod`                   | `istio-system` | `53`    | UDP/TCP   | Egress    | DNS resolution                                                                               |
| `istiod`                   | `istio-system` | `80`    | TCP       | Egress    | Access to external JWKS endpoints for JWT validation (HTTP)                                  |
| `istiod`                   | `istio-system` | `443`   | TCP       | Egress    | Access to external JWKS endpoints for JWT validation (HTTPS) / Kubernetes API server access |
| `istiod`                   | `istio-system` | `15012` | TCP/gRPC  | Ingress   | XDS config distribution to sidecars and gateways                                             |
| `istiod`                   | `istio-system` | `15014` | TCP/HTTP  | Ingress   | Control plane metrics (Prometheus scrape)                                                    |
| `istiod`                   | `istio-system` | `15017` | TCP/HTTPS | Ingress   | Webhook endpoint (defaulting/mutation/admission)                                             |
| `istio-ingressgateway`     | `istio-system` | `*`     | TCP       | Egress    | Traffic to labeled user Pods (`networking.kyma-project.io/from-ingressgateway: allowed`)     |
| `istio-ingressgateway`     | `istio-system` | `53`    | UDP/TCP   | Egress    | DNS resolution                                                                               |
| `istio-ingressgateway`     | `istio-system` | `8080`  | TCP       | Ingress   | HTTP traffic inbound to cluster                                                              |
| `istio-ingressgateway`     | `istio-system` | `8443`  | TCP       | Ingress   | HTTPS traffic inbound to cluster                                                             |
| `istio-ingressgateway`     | `istio-system` | `15008` | TCP       | Ingress   | HBONE mTLS tunnel (Ambient mode)                                                             |
| `istio-ingressgateway`     | `istio-system` | `15012` | TCP/gRPC  | Egress    | Request XDS config from istiod                                                               |
| `istio-ingressgateway`     | `istio-system` | `15020` | TCP/HTTP  | Ingress   | Merged Prometheus metrics                                                                    |
| `istio-ingressgateway`     | `istio-system` | `15021` | TCP/HTTP  | Ingress   | Health check endpoint                                                                        |
| `istio-ingressgateway`     | `istio-system` | `15090` | TCP/HTTP  | Ingress   | Envoy Prometheus telemetry                                                                   |
| `istio-cni-node`           | `istio-system` | `53`    | UDP/TCP   | Egress    | DNS resolution                                                                               |
| `istio-cni-node`           | `istio-system` | `443`   | TCP       | Egress    | Kubernetes API server access                                                                 |

Enabling the **enableModuleNetworkPolicies** flag creates the necessary NetworkPolicies to allow traffic for Istio components.

- Allow APIServer and DNS access for `istio-controller-manager` in `kyma-system`:

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: NetworkPolicy
  metadata:
    labels:
      kyma-project.io/module: istio
      kyma-project.io/managed-by: kyma
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

- Allow the following traffic for `istiod`:
  - Allow egress access to DNS and APIServer.
  - Allow ingress access from sidecars and `istio-ingressgateway` on port `15012` for control-plane communication.
  - Allow ingress access to the control plane monitoring port `15014`.
  - Allow ingress access to the webhook endpoint on port `15017`.

    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: NetworkPolicy
    metadata:
      labels:
        kyma-project.io/module: istio
        kyma-project.io/managed-by: kyma
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
          # The namespaceSelector needs to be set to empty `{}` explicitly.
          # In case it is not specified, the policy will only allow traffic from the same namespace.
          namespaceSelector: {}
        - podSelector:
            matchLabels:
              istio: ingressgateway
        ports:
        - protocol: TCP
          port: 15012
      - ports:
        - protocol: TCP
          port: 15014
        - protocol: TCP
          port: 15017
    ```

- Allow the following traffic for `istio-ingressgateway`:
  - Allow egress access to `istiod` on port `15012` (XDS).
  - Allow egress access to DNS.
  - Allow ingress access from external traffic on ports `80`/`443`.
  - Allow ingress access to the HBONE mTLS tunnel port (`15008`), merged Prometheus metrics port (`15020`), health checks (`15021`), and Envoy Prometheus telemetry (`15090`).

    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: NetworkPolicy
    metadata:
      labels:
        kyma-project.io/module: istio
        kyma-project.io/managed-by: kyma
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
        - protocol: TCP
          port: 15008
        - protocol: TCP
          port: 15020
        - protocol: TCP
          port: 15021
        - protocol: TCP
          port: 15090
    ```

- Allow egress access to DNS and APIServer for `istio-cni-node`:
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: NetworkPolicy
  metadata:
    labels:
      kyma-project.io/module: istio
      kyma-project.io/managed-by: kyma
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

- Allow access to external (outside of cluster) JWKS endpoints for JWT verification by `istiod`:

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: NetworkPolicy
  metadata:
    labels:
      kyma-project.io/module: istio
      kyma-project.io/managed-by: kyma
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

- Allow user-enabled egress traffic from `istio-ingressgateway` to backend services by creating a NetworkPolicy
that allows egress to specifically labeled Pods in user namespaces:

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: NetworkPolicy
  metadata:
    labels:
      kyma-project.io/module: istio
      kyma-project.io/managed-by: kyma
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
        # The namespaceSelector needs to be set to empty `{}` explicitly.
        # In case it is not specified, the policy will only allow traffic from the same namespace.
        namespaceSelector: {}
  ```

### Restart Istio Components to Enforce Policies
To ensure that the newly applied NetworkPolicies take effect immediately, all Istio components must be restarted (rollout restart)
when the user enables the NetworkPolicy support flag in the Istio CR.
Enabling the flag terminates any existing TCP connections and allows the new policies to be enforced without delay.
This means that a restart should happen whenever the value of **enableModuleNetworkPolicies** is changed from `false` to `true` or vice versa,
to ensure that the correct policies are applied based on the user's choice.

### User Impact

To ensure that the connectivity between `istio-ingressgateway` and user workloads is maintained when NetworkPolicy support is enabled,
users must label the Pods in their namespaces that should be accessible from `istio-ingressgateway`
using the label `networking.kyma-project.io/from-ingressgateway: allowed`.

### Default-Deny Policies in User Namespaces
If the user desires to use the default-deny policy in their namespaces,
they must create appropriate NetworkPolicies to allow traffic from the `istio-ingressgateway` to their workloads. 
Applying these NetworkPolicies will not be automatically handled by the Istio module,
as they must be applied outside the `istio-system` or `kyma-system` namespaces. This action is required for users who want to enable NetworkPolicy support and must be documented.
