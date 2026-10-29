# Set a Custom Error Status Code for an External Authorizer

Learn how to configure the **statusOnError** field of an external authorizer in the Istio custom resource (CR), so that Istio returns a custom HTTP status code when it cannot reach the external authorization service.

## Context

When you protect a workload with an external authorizer, every request is first sent to the external authorization service for a decision. If Istio cannot reach that service because of a network error, it returns a status code to the client instead of the authorization decision. By default, this status code is `403`.

The **statusOnError** field overrides this default. You set it on an authorizer in the Istio CR, and Istio returns the configured status code whenever the external authorization service is unreachable.

### The statusOnError Field

The **statusOnError** field defines the HTTP status code that Istio returns to the client when a network error to the external authorization service occurs. It applies only to network errors, not to a regular deny decision from the authorization service.

The field accepts a string that contains a 4xx or 5xx HTTP status code, for example, `"417"`. The following rules apply:

- The value must be a three-digit status code in the 4xx or 5xx range. Other values are rejected when you apply the Istio CR.
- The value must be the numeric status code, such as `"403"`, not the name of the code, such as `FORBIDDEN`.
- The status code must be one of the codes defined in Envoy's [`type.v3.StatusCode`](https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto) enum. Envoy ignores a code that is outside this enum, such as `418`, and falls back to the default status code `403`.
- If you don't set the field, Istio uses the default status code `403`.

### Methods of Configuring an External Authorizer

You can point an authorizer at the external authorization service in two ways:

- **In-cluster workload**: The authorizer's **service** field references a Kubernetes Service that runs in your cluster. Use this method when the authorization service runs as a workload in the mesh.
- **ServiceEntry and DestinationRule**: The authorizer's **service** field references a host that you register in Istio's service registry with a ServiceEntry. A DestinationRule configures how Envoy connects to that host. Use this method when the authorization service runs outside the cluster.

This tutorial uses an authorization service that is deliberately unreachable, so you can observe the **statusOnError** fallback. It shows both configuration methods.

## Prerequisites

- You have the Istio module installed and deployed.
- To expose the workload, you use one of the following:
    - The default domain of your Kyma cluster and the default Gateway `kyma-system/kyma-gateway`. This Gateway is available only when you add the API Gateway module.
    - A custom domain and a custom TLS Gateway.

## Procedure

### Expose a Sample Workload

1. Export the name of the namespace in which you want to deploy the sample workload:

    ```bash
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create the namespace with Istio sidecar injection enabled, and deploy a sample HTTPBin workload:

    ```bash
    kubectl create namespace $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl apply --namespace $NAMESPACE --filename https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

3. Export the name of your domain and Gateway:

    ```bash
    export DOMAIN_NAME={YOUR_DOMAIN_NAME}
    export GATEWAY={YOUR_GATEWAY}
    ```

4. Create a VirtualService to expose the workload:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: networking.istio.io/v1
    kind: VirtualService
    metadata:
      name: httpbin
      namespace: $NAMESPACE
    spec:
      hosts:
        - "httpbin.$DOMAIN_NAME"
      gateways:
        - $GATEWAY
      http:
        - match:
            - uri:
                prefix: /
          route:
            - destination:
                host: httpbin.$NAMESPACE.svc.cluster.local
                port:
                  number: 8000
    EOF
    ```

### Configure the External Authorizer

Choose one of the following methods to register the external authorizer. Both methods point the authorizer at an unreachable authorization service and set **statusOnError** to `"417"`.

#### Method 1: Reference an In-Cluster Workload

Deploy a sample authorization service, and then register an authorizer whose **service** field references its Kubernetes Service.

> ### Note:
> The Kubernetes Service that the **service** field references must be deployed in your cluster. If the Service doesn't exist, Istio returns the default status code `403` regardless of the **statusOnError** setting. In this tutorial, the authorization call fails because the **port** is unreachable, not because the Service is missing.

1. Export the name of the namespace for the authorization service:

    ```bash
    export AUTH_NAMESPACE={AUTH_NAMESPACE_NAME}
    ```

2. Create the namespace with Istio sidecar injection enabled:

    > ### Note:
    > The authorization service must run in a namespace that has Istio sidecar injection enabled. Without the injected sidecar, the external authorization check isn't applied.

    ```bash
    kubectl create namespace $AUTH_NAMESPACE
    kubectl label namespace $AUTH_NAMESPACE istio-injection=enabled --overwrite
    ```

3. Deploy a sample HTTPBin workload as the authorization service:

    ```bash
    kubectl apply --namespace $AUTH_NAMESPACE --filename https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

4. Register an authorizer whose **service** field references the HTTPBin Service. Set the **port** to a port on which no workload listens, so the authorization call fails with a network error:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: operator.kyma-project.io/v1alpha2
    kind: Istio
    metadata:
      name: default
      namespace: kyma-system
    spec:
      config:
        authorizers:
          - name: httpbin-authorizer
            service: "httpbin.$AUTH_NAMESPACE.svc.cluster.local"
            port: 9999
            timeout: 2s
            statusOnError: "417"
    EOF
    ```

#### Method 2: Reference a ServiceEntry and DestinationRule

Register an authorizer whose **service** field references a host that exists only in Istio's service registry. Because the host doesn't resolve to a running service, the authorization call fails with a network error.

1. Create a ServiceEntry and a DestinationRule for the external host:

    > ### Note:
    > The host `ext-authz.example.invalid` uses the reserved `.invalid` top-level domain, so it never resolves to a real service.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: networking.istio.io/v1
    kind: ServiceEntry
    metadata:
      name: ext-authz
      namespace: kyma-system
    spec:
      hosts:
        - ext-authz.example.invalid
      location: MESH_EXTERNAL
      resolution: DNS
      ports:
        - number: 9999
          name: http
          protocol: HTTP
    ---
    apiVersion: networking.istio.io/v1
    kind: DestinationRule
    metadata:
      name: ext-authz
      namespace: kyma-system
    spec:
      host: ext-authz.example.invalid
      trafficPolicy:
        connectionPool:
          tcp:
            connectTimeout: 1s
    EOF
    ```

2. Register the authorizer with the ServiceEntry host:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: operator.kyma-project.io/v1alpha2
    kind: Istio
    metadata:
      name: default
      namespace: kyma-system
    spec:
      config:
        authorizers:
          - name: httpbin-authorizer
            service: "ext-authz.example.invalid"
            port: 9999
            timeout: 2s
            statusOnError: "417"
    EOF
    ```

### Activate the Authorizer

1. Wait until the Istio CR is in the `Ready` state:

    ```bash
    kubectl wait --for=jsonpath='{.status.state}'=Ready istio/default --namespace kyma-system --timeout=180s
    ```

2. Create an AuthorizationPolicy CR with the CUSTOM action that references the authorizer:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: security.istio.io/v1
    kind: AuthorizationPolicy
    metadata:
      name: ext-authz
      namespace: $NAMESPACE
    spec:
      action: CUSTOM
      provider:
        name: httpbin-authorizer
      rules:
        - to:
            - operation:
                paths: ["/*"]
    EOF
    ```

### Results

Send a request to the exposed workload:

```bash
curl -s -o /dev/null -w "%{http_code}\n" "https://httpbin.$DOMAIN_NAME/headers"
```

The authorizer cannot reach the authorization service, so Istio applies the **statusOnError** fallback. The request returns the status code `417` instead of the default `403`.

> ### Note:
> The CUSTOM-provider configuration takes some time to propagate to the sidecar proxy. If the request returns `200`, wait a moment, and then try again.
