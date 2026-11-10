# Network Connectivity Diagnostics

If you're having trouble with network connectivity and don't know where to begin troubleshooting, follow these steps. The issues might not be directly related to Istio, but could be caused by misconfigured Istio resources or other cluster resources.

> ### Note:
> Before you start, install the required tools and get familiar with the Istio architecture. For more information, see [Prerequisites and Architecture](./03-00-prerequisites.md).

- Verify the state of the Istio CR. If it is in the `Warning` state, check the warning message and conditions. It might help you begin the investigation.
- Verify that no [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) affect the connectivity by blocking traffic between Pods in the service mesh. To find all NetworkPolicy resources, run `kubectl get networkpolicies --all-namespaces`.
- The configuration of the following resources can affect the connectivity in the service mesh. Verify that these resources are configured as intended:
    - [DestinationRule](https://istio.io/latest/docs/reference/config/networking/destination-rule/)
    - [PeerAuthentication](https://istio.io/latest/docs/reference/config/security/peer_authentication/)
    - [Gateway](https://istio.io/latest/docs/reference/config/networking/gateway/)
    - [AuthorizationPolicy](https://istio.io/latest/docs/reference/config/security/authorization-policy/)
    - [RequestAuthentication](https://istio.io/latest/docs/reference/config/security/request_authentication/)

## Starting Point

Begin with two quick checks that narrow down the problem. First, run `istioctl analyze` to find configuration errors in the mesh. Then, send a `curl` request to the failing endpoint to see how far it gets and which component responds.

### Analyze the Mesh Configuration

Run `istioctl analyze` to detect common errors and misconfigurations in your mesh.

- Analyze a specific namespace:
  ```bash
  istioctl analyze --namespace {NAMESPACE}
  ```

- Analyze the entire cluster:
  ```bash
  istioctl analyze --all-namespaces
  ```

If the analyzer doesn't find any problems, you get the following output:

```
✔ No validation issues found when analyzing namespace: default.
```

If the analyzer finds issues, it prints one line per finding. Each line contains the following information:

- The severity: `Error`, `Warning`, or `Info`
- The message code, such as `IST0133`
- The kind, namespace, and name of the affected resource
- A description of the issue

See the following example output:

```
Warning [IST0133] (AuthorizationPolicy app-403/deny-no-port) Schema validation warning: configured AuthorizationPolicy will deny all traffic to TCP ports under its scope due to the use of only HTTP attributes in a DENY rule; it is recommended to explicitly specify the port
Error [IST0145] (Gateway default/app-404-gateway) Conflict with gateways team-b/other-gateway (workload selector istio=ingressgateway, port 80, hosts app-404.example.com).
```

Review all output carefully and use the message code and the named resource to decide what to check next. The analyzer reports the resources using the `(Kind namespace/name)` format. For example, the previous output shows the following issues:

- `IST0133`: A `DENY` AuthorizationPolicy uses only HTTP attributes and might block more TCP traffic than intended. See [AuthorizationPolicies](#authorizationpolicies).
- `IST0145`: Two Gateway resources claim the same host, which causes `404` errors. See [Diagnosing 404 Errors](./03-12-404-diagnostics.md).

For any other finding, open the resource named in the message and verify its configuration.

A clean `istioctl analyze` result does not guarantee that traffic flows. It only means that the mesh configuration is internally consistent. Continue with the `curl` check.

### Send a Request to the Failing Endpoint

Send a request to the endpoint that fails and include the response headers. The `-D -` flag prints the response headers. They reveal whether the request reached Istio Ingress Gateway and which component produced the response:

```bash
curl -sD - http://{YOUR_HOST}/{YOUR_PATH}
```

If you access the service mesh through a custom domain, first confirm that the hostname resolves to the external IP of the `istio-ingressgateway` Service. A `curl` failure caused by a broken DNS record looks like a connectivity problem, but no request ever leaves your machine. Use `nslookup` or `dig` to check the record:

```bash
nslookup {YOUR_HOST}
```

If the returned address is the external IP of the `istio-ingressgateway` Service, DNS is correct. Continue with the `curl` analysis. If the command returns a different address, or reports `NXDOMAIN`, the hostname does not resolve, and no request reaches the mesh. Fix the DNS record first. For the full procedure, see [Validate the DNS Resolution](#validate-the-dns-resolution).

Analyze the `curl` output to decide where to continue:

- **No HTTP response at all.** The command reports a connection timeout or `Connection refused`, and there is no `server` header because no HTTP exchange happened. The traffic is blocked before it reaches Istio Ingress Gateway, either because the hostname does not resolve or because a network-layer (L3/L4) block, such as an external firewall, drops the packets. See [Connection Does Not Reach the Service Mesh](#connection-does-not-reach-the-service-mesh).

  ```
  curl: (28) Failed to connect to {YOUR_HOST} port 80 after 5002 ms: Couldn't connect to server
  ```

- **An HTTP response with no `server: istio-envoy` header, or a non-Envoy `server` value.** An external proxy, firewall, or WAF in front of the cluster produced the response before it reached Istio. This is not an Istio problem. See [Check for an External Firewall Blocking Access](#check-for-an-external-firewall-blocking-access).

- **An HTTP response with the `server: istio-envoy` header.** The request reached Istio Ingress Gateway. Use the status code to choose the next step:

  ```
  HTTP/1.1 404 Not Found
  server: istio-envoy
  content-length: 0
  ```

  | Status code | Meaning | Where to continue |
  |---|---|---|
  | `200` | The request succeeded and reached the workload. | The connectivity path works. If a later request still fails, check the specific Istio resource that handles it in [Debugging Istio Resources](#debugging-istio-resources). |
  | `401 Unauthorized` | A RequestAuthentication rejected the request, typically because of a missing or invalid JWT. | [RequestAuthentications](#requestauthentications) |
  | `403 Forbidden` | An AuthorizationPolicy, your application, or an external component rejected the request. | [Diagnosing 403 Errors](./03-11-403-diagnostics.md) |
  | `404 Not Found` | No route matched the host and path, two Gateway resources conflict, or your application returned the error. | [Diagnosing 404 Errors](./03-12-404-diagnostics.md) |
  | `503 Service Unavailable` | Istio Ingress Gateway could not reach a healthy upstream, or your application returned the error. | [Diagnosing 503 Errors](./03-10-503-diagnostics.md) |

## Connection Does Not Reach the Service Mesh

Several issues can affect external connectivity to the service mesh. Follow these steps to verify that the cluster endpoint is reachable and working.

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

The cloud provider typically performs this configuration, which is not part of Istio. If you have this problem, contact your cloud provider for support.

### Check the Istio Gateway Resource

The Gateway resource defines the ports and protocols that the `istio-ingressgateway` Service listens on. If the Gateway resource is misconfigured, the service mesh is not reachable. 

In SAP BTP, Kyma runtime, the API Gateway module creates and manages the default Gateway resource `kyma-gateway` in the `kyma-system` namespace. If you create your own Gateway resources, you're responsible for their configuration.

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

This configuration is not part of Istio, and the DNS provider typically handles it. If the hostname doesn't resolve correctly, contact your DNS provider.

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

If the DNS resolves correctly to the external IP but the connection still fails, the traffic might be blocked **before** it reaches Istio Ingress Gateway. The most common cause is a firewall outside the cluster, such as a cloud
provider security group, a network ACL, or a corporate firewall between you
and the cluster. This is a network-layer (L3/L4) block and is not part of Istio.

The key symptom of an external firewall block is that the connection never completes. There's no HTTP response at all, so there's no `server: istio-envoy` header either. Instead, you get a TCP-level failure, such as a timeout or a connection refused error. Compare the following two outcomes.

- A connection timeout typically means that a firewall silently drops the packets:

    

To find out whether the port itself is reachable, independently of HTTP, test the TCP connection directly. For example, test port 80 and the HTTPS port 443:

```bash
nc -zv host.my-service.example.com 80
nc -zv host.my-service.example.com 443
```

If `nc` cannot open the connection, but the `istio-ingressgateway` Service and the Gateway resource are both configured correctly, the block is almost certainly external to the cluster. Verify the following conditions are met:

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