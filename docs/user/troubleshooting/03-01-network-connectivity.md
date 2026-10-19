# Network Connectivity - Diagnostics

If you're having trouble with network connectivity and don't know where to begin troubleshooting, follow these steps. The issues might not be directly related to Istio, but could be caused by misconfigured Istio resources or other cluster resources.

> ### Note:
> Before you start, install the required tools and get familiar with the Istio architecture. For more information, see [Network Connectivity - Prerequisites and Architecture](./03-00-prerequisites.md).

- Verify the state of the Istio CR. If it is in the `Warning` state, check the warning message and conditions. It might help you begin the investigation.
- Verify that no [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) affect the connectivity by blocking traffic between Pods in the service mesh. To find all NetworkPolicy resources, run `kubectl get networkpolicies --all-namespaces`.
- The configuration of the following resources can affect the connectivity in the service mesh. Verify that these resources are configured as intended:
    - [DestinationRule](https://istio.io/latest/docs/reference/config/networking/destination-rule/)
    - [PeerAuthentication](https://istio.io/latest/docs/reference/config/security/peer_authentication/)
    - [Gateway](https://istio.io/latest/docs/reference/config/networking/gateway/)
    - [AuthorizationPolicy](https://istio.io/latest/docs/reference/config/security/authorization-policy/)
    - [RequestAuthentication](https://istio.io/latest/docs/reference/config/security/request_authentication/)

## Starting Point

As a starting point, run `istioctl analyze` to check the state of your mesh and point out common errors and misconfigurations.

- Analyze a specific namespace:
  ```bash
  istioctl analyze --namespace {NAMESPACE}
  ```

- Analyze the entire cluster:
  ```bash
  istioctl analyze --all-namespaces
  ```

Review all output carefully, especially warnings such as `IST0133`, which indicate that an AuthorizationPolicy might be blocking more traffic than intended. The analysis often includes the name and namespace of the problematic resource.

## Connection Does Not Reach the Service Mesh

Several things can cause issues with external connectivity to the service mesh. Follow these steps to verify that the cluster endpoint is reachable and working.

### Check the Istio Ingress Gateway Service

Istio Ingress Gateway is the entrypoint to the service mesh. It is deployed with a Kubernetes Service of type `LoadBalancer`, which exposes Istio Ingress Gateway to the external world with a dedicated IP address that the cloud provider assigns.

List the `istio-ingressgateway` Service and check the external IP address:

```bash
kubectl get svc istio-ingressgateway --namespace istio-system
```
```
NAME                   TYPE           CLUSTER-IP        EXTERNAL-IP                                                                        PORT(S)                                      AGE
istio-ingressgateway   LoadBalancer   100.108.172.136   REDACTED.elb.eu-central-2.amazonaws.com   15021:32069/TCP,80:31572/TCP,443:30745/TCP   85s
```

If the external IP is `<pending>`, the Service is not yet reachable from the external world. If this state persists for a longer time, check the cloud provider configuration and verify that you are not missing any additional settings or permissions. Without a valid external IP, you cannot reach the service mesh.

The cloud provider usually performs this configuration, which is not part of Istio. If you have this problem, contact your cloud provider for support.

### Check the Istio Gateway Resource

The Gateway resource defines the ports and protocols that the `istio-ingressgateway` Service listens on. If the Gateway resource is misconfigured, the service mesh is not reachable. The user creates this resource and is responsible for any changes to it.

> ### Note:
> If you use SAP BTP, Kyma runtime, the API Gateway module manages the Gateway resource `kyma-gateway`. However, the same troubleshooting steps apply.

List all Gateway resources in a namespace:
```bash
kubectl get gateways --namespace {NAMESPACE}
```
```
NAMESPACE            NAME             AGE
{NAMESPACE}          istio-gateway    59s
```

The Gateway resource defines the ports and hosts that the cluster endpoint listens on. The most important fields are `spec.servers` and `spec.selector`. The `spec.servers` field defines the ports and protocols that the `istio-ingressgateway` Service listens on. The `spec.selector` field defines which `istio-ingressgateway` Service is used. If the selector does not match the labels of the `istio-ingressgateway` Service, the Gateway resource does not work.

Validate that the `spec.selector` field of the Gateway resource matches the labels of the `istio-ingressgateway` Service. To check the labels of the `istio-ingressgateway` Service, run the following command:

```bash
kubectl get svc istio-ingressgateway --namespace istio-system --output jsonpath='{.metadata.labels}'
```
```
{
(...)
"istio": "ingressgateway",
(...)
}
```
```bash
kubectl get gateways.networking.istio.io --namespace default simple-gateway --output jsonpath='{.spec.selector}'
```
```
{"istio":"ingressgateway"}
```

If the selector does not match, update the Gateway resource to use the correct labels.

Describe the Gateway resource to see its configuration:
```bash
kubectl describe gateway istio-gateway --namespace {NAMESPACE}
```
```
Name:         istio-gateway
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  networking.istio.io/v1
Kind:         Gateway
Metadata:
  Creation Timestamp:  2026-10-20T11:25:20Z
  Generation:          1
  Resource Version:    12614
  UID:                 c5b35a89-7408-4d37-b624-e78b9f1af583
Spec:
  Selector:
    Istio:  ingressgateway
  Servers:
    Hosts:
      *
    Port:
      Name:      http
      Number:    80
      Protocol:  HTTP
Events:          <none>
```

In this example, the Gateway resource is configured to listen on port 80 for all hosts. If you use the external IP of the `istio-ingressgateway` Service, you can reach the service mesh on port 80.

Verify that the `spec.servers` field of the Gateway resource is configured correctly. The most common misconfigurations are the following:

- The `hosts` field is invalid or not reachable from the external world.
- The `port` field is not set to the correct port.

If you see any misconfiguration, update the Gateway resource to use the correct host and port.

### Validate the DNS Resolution

If you use a hostname in the Gateway resource, verify that the hostname resolves to the external IP of the `istio-ingressgateway` Service. To check the DNS resolution, use the `nslookup` or `dig` command. For example, if your Gateway resource is configured to listen on `host.my-service.example.com`, run the following command:

```bash
nslookup host.my-service.example.com
```
```
Server:		8.8.8.8
Address:	8.8.8.8#53

Non-authoritative answer:
Name:	host.my-service.example.com
Address: XXX.XXX.XXX.XXX
```

If you get a different IP address than the external IP of the `istio-ingressgateway` Service, or if you get an `NXDOMAIN` error, the hostname does not resolve correctly. Update the DNS records to point to the correct external IP of the `istio-ingressgateway` Service.

This configuration is not part of Istio, and the DNS provider usually performs it. If you have this problem, contact your DNS provider for support.

If you use Gardener, you can manage DNS records with the Gardener DNS extension. For more information, see the [Gardener documentation](https://gardener.cloud/docs/extensions/others/gardener-extension-shoot-dns-service/#gardener-extension-for-dns-services).

For Gardener, verify that your DNSEntry resources are present and in the `Ready` state.

> ### Note:
> If you use SAP BTP, Kyma runtime, the API Gateway module manages the default DNSEntry resource. However, the same troubleshooting steps apply.

To check the status of the DNSEntry resources, run the following command:

```bash
kubectl get dnsentries --namespace {NAMESPACE}
```
```
NAMESPACE   NAME                DNS                                           TYPE     PROVIDER                      STATUS   AGE   TARGETS
default     wildcard-dnsentry   *.REDACTED.shoot.live.k8s-hana.ondemand.com   remote   remote: shoot--xxx/external   Ready    14m   ["REDACTED.elb.eu-central-2.amazonaws.com"]
```

Verify the following values:
- The `TARGETS` field matches the external IP of the `istio-ingressgateway` Service.
- The `STATUS` field is `Ready`. If it is not `Ready`, the DNS entry is not yet propagated.
- The `DNS` field matches the `hosts` field of the Gateway resource.

If any of the fields do not match the expected values, update the DNSEntry resources to use the correct values.

After you verify that everything is configured correctly, you can reach the service mesh with the hostname:

```bash
curl -Lv http://host.my-service.example.com
```
```
* Host host.my-service.example.com:80 was resolved.
* IPv6: (none)
* IPv4: XXX.XXX.XXX.XXX
*   Trying XXX.XXX.XXX.XXX:80...
* Connected host.my-service.example.com (XXX.XXX.XXX.XXX) port 80
> GET / HTTP/1.1
> Host: host.my-service.example.com
> User-Agent: curl/8.7.1
> Accept: */*
>
* Request completely sent off
< HTTP/1.1 404 Not Found
< date: Tue, 20 Oct 2026 12:29:56 GMT
< server: istio-envoy
< content-length: 0
<
* Connection #0 to host host.my-service.example.com left intact
```

Check the following signals in the response:
- The `server: istio-envoy` header indicates that the request reached Istio Ingress Gateway and the Envoy proxy processed it.
- The `HTTP/1.1 404 Not Found` response indicates that the request reached the service mesh, but no route matched the requested host and path. This is expected if you have not configured any VirtualService resources for the requested host and path. To find out whether incomplete routing, a Gateway host conflict, or your application causes the error, see [Diagnosing 404 Errors](./03-12-404-diagnostics.md).

### Check for an External Firewall Blocking Access

If the DNS resolves correctly to the external IP but the connection still fails, the traffic might be blocked **before** it reaches Istio Ingress Gateway. A firewall outside of the cluster typically causes this, such as a cloud provider security group, a network ACL, or a corporate firewall between you and the cluster. This is a network-layer (L3/L4) block and is not part of Istio.

The key symptom of an external firewall block is that the connection never completes. There is no HTTP response at all, and therefore no `server: istio-envoy` header. Instead, you see a TCP-level failure, such as a timeout or a connection refused. Compare the following two outcomes.

A connection timeout usually means that a firewall silently drops the packets:

```bash
curl -Lv --connect-timeout 5 http://host.my-service.example.com
```
```
*   Trying XXX.XXX.XXX.XXX:80...
* connect to XXX.XXX.XXX.XXX port 80 failed: Operation timed out
* Failed to connect to host.my-service.example.com port 80 after 5002 ms: Couldn't connect to server
* Closing connection
curl: (28) Failed to connect to host.my-service.example.com port 80 after 5002 ms: Couldn't connect to server
```

A `Connection refused` error usually means that nothing is listening on that port, or a firewall actively rejects the packets:

```
*   Trying XXX.XXX.XXX.XXX:80...
* connect to XXX.XXX.XXX.XXX port 80 failed: Connection refused
```

To find out whether the port itself is reachable, independently of HTTP, test the TCP connection directly. For example, test port 80 and the HTTPS port 443:

```bash
nc -zv host.my-service.example.com 80
nc -zv host.my-service.example.com 443
```

If `nc` cannot open the connection, but the `istio-ingressgateway` Service and the Gateway resource are both configured correctly, the block is almost certainly external to the cluster. Verify the following:

- The `istio-ingressgateway` Service exposes the port that you test. Only the ports listed in the Service are reachable through the load balancer:
  ```bash
  kubectl get svc istio-ingressgateway --namespace istio-system --output jsonpath='{range .spec.ports[*]}{.name}{": "}{.port}{"\n"}{end}'
  ```
  ```
  status-port: 15021
  http2: 80
  https: 443
  ```
- The cloud provider security group or firewall rules attached to the load balancer allow inbound traffic on that port from your source IP range.
- There is no corporate or VPN firewall between you and the cluster that blocks the port.

The cloud provider or your network administrator performs this configuration, which is not part of Istio. If the port is blocked externally, contact your cloud provider or network administrator for support.

A firewall, proxy, or WAF can also return an HTTP `403 Forbidden` response, which looks similar to a `403` error from an AuthorizationPolicy inside the mesh. To tell the two apart and find the right fix, see [Diagnosing 403 Errors](./03-11-403-diagnostics.md).

## Debugging Istio Resources

If you have confirmed that Istio Ingress Gateway is reachable and returns a response, but you still experience connectivity issues, check the Istio resources that control traffic flow.

### AuthorizationPolicies

Misconfigured AuthorizationPolicy resources are a common cause of access issues, such as `403` errors or unexpected connection timeouts.

- List all AuthorizationPolicy resources in a namespace:
  ```bash
  kubectl get authorizationpolicies --namespace {NAMESPACE}
  ```
- List all AuthorizationPolicy resources cluster-wide:
  ```bash
  kubectl get authorizationpolicies --all-namespaces
  ```
- Describe a specific AuthorizationPolicy to see its details:
  ```bash
  kubectl describe authorizationpolicy {POLICY_NAME} --namespace {NAMESPACE}
  ```

Check the following:
- **Deny policies:** Deny policies might block more traffic than intended. A deny rule without the **ports** field on an HTTP-based **hosts** rule can block all TCP traffic, not just HTTP.
- **Allow policies:** If even a single allow policy matches a workload, all traffic that does not match any allow rule is denied. Verify that all intended traffic is covered by allow rules.
- **Scope:** Check the **selector** to see which workloads the policy applies to. An empty selector applies the policy to all workloads in the namespace.
- **Rules:** Look at the **to**, **from**, and **when** clauses to understand which traffic is allowed or denied.

### RequestAuthentications

RequestAuthentication defines how JSON Web Tokens (JWTs) are validated. Misconfigurations often lead to `401 Unauthorized` responses.

- List all RequestAuthentication resources in a namespace:
  ```bash
  kubectl get requestauthentications --namespace {NAMESPACE}
  ```
- Describe a specific RequestAuthentication to see its configuration:
  ```bash
  kubectl describe requestauthentication {REQUEST_AUTHENTICATION_NAME} --namespace {NAMESPACE}
  ```

Check the following:
- **jwksUri:** Verify that the JSON Web Key Set (JWKS) endpoint is correct and reachable from `istiod`.
- **issuer:** The `iss` claim in the JWT must match this value.
- **audiences:** If these are set, the `aud` claim in the JWT must match one of the values here.

### EnvoyFilters

EnvoyFilter resources modify the Envoy configuration and can easily cause problems if misconfigured. They are often used for complex use cases.

- List all EnvoyFilter resources cluster-wide:
  ```bash
  kubectl get envoyfilters --all-namespaces
  ```
- Examine a specific filter:
  ```bash
  kubectl get envoyfilter {FILTER_NAME} --namespace {NAMESPACE} --output yaml
  ```

Check the following:
- **Workload selector:** Check which Pods the filter applies to.
- **Filters:** Look at the filter configuration. These are often complex Lua scripts or WebAssembly (WASM) filters. A bug in the script can break traffic.

### Envoy Configuration

If the previous steps do not reveal the issue, inspect the live Envoy configuration for a specific Pod. This shows exactly which rules Envoy applies.

- Get the proxy configuration for a specific Pod:
  ```bash
  istioctl proxy-config <all|listeners|clusters|routes|endpoints|bootstrap> {POD_NAME} --namespace {NAMESPACE}
  ```
- For example, check the routes for a Pod:
  ```bash
  istioctl proxy-config routes {POD_NAME}.{NAMESPACE} --output json
  ```