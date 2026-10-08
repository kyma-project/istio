# Exposing Workloads Using Gateway API 

The Istio module installs the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs. This tutorial shows how to expose an HTTP workload using a Gateway resource and an HTTPRoute.

## Prerequisites

* You have the Istio module added.

## Context

In this tutorial, you expose a sample [HTTPBin Service](https://httpbin.org/) outside the cluster using the Kubernetes Gateway API. First, you create a [Gateway](https://gateway-api.sigs.k8s.io/reference/api-types/gateway/) resource that defines the entry point for external traffic. Then, you create an [HTTPRoute](https://gateway-api.sigs.k8s.io/reference/api-types/httproute/) that defines how incoming traffic is forwarded to the HTTPBin Service. Istio acts as the Gateway API controller and provisions the infrastructure needed to route traffic. To verify the setup, you call the service and confirm it returns a `200 OK` response.

For details on Gateway API CRD management in the Istio module, see [Gateway API CRDs Management](../00-55-gateway-api-crds.md).

## Procedure

1. Export the name of the namespace in which you want to deploy the HTTPBin Service:
    ```bash
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the HTTPBin Service.

    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.31/samples/httpbin/httpbin.yaml
    ```

    Verify all Pods are running:

    ```bash
    kubectl get pods -l app=httpbin -n $NAMESPACE
    ```

3. To create an entry point for external HTTP traffic, create a Kubernetes Gateway resource:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: httpbin-gateway
      namespace: ${NAMESPACE}
    spec:
      gatewayClassName: istio
      listeners:
      - name: http
        hostname: "httpbin.kyma.example.com"
        port: 80
        protocol: HTTP
        allowedRoutes:
          namespaces:
            from: Same
    EOF
    ```

    `httpbin.kyma.example.com` is a placeholder hostname used for this tutorial. Istio detects the new Gateway resource and automatically provisions a dedicated Envoy proxy pod and a Kubernetes Service of type `LoadBalancer` in your namespace to handle incoming traffic.

    Verify that the Gateway is programmed and has an external address assigned — this confirms the LoadBalancer is ready to accept traffic:

    ```bash
    kubectl get gtw httpbin-gateway -n $NAMESPACE
    ```

4. Create an HTTPRoute to configure access to your workload:

    The HTTPRoute exposes only the `/headers` endpoint of the HTTPBin Service, which returns the headers of the incoming request. The HTTPBin Service listens on port `8000`.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: HTTPRoute
    metadata:
      name: httpbin
      namespace: ${NAMESPACE}
    spec:
      parentRefs:
      - name: httpbin-gateway
      hostnames: ["httpbin.kyma.example.com"]
      rules:
      - matches:
        - path:
            type: PathPrefix
            value: /headers
        backendRefs:
        - name: httpbin
          namespace: ${NAMESPACE}
          port: 8000
    EOF
    ```

    Check that the route is valid and all references are resolved:

    ```bash
    kubectl describe httproute httpbin -n $NAMESPACE
    ```

5. To verify access to the HTTPBin Service, follow the steps:

    1. Discover the gateway's external address and port:
        
        ```bash
        export INGRESS_HOST=$(kubectl get gtw httpbin-gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
        export INGRESS_PORT=$(kubectl get gtw httpbin-gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="http")].port}')
        echo "Ingress host: $INGRESS_HOST, port: $INGRESS_PORT"
        ```

    2. Call the service:
        
        ```bash
        curl -s -I -HHost:httpbin.kyma.example.com "http://$INGRESS_HOST:$INGRESS_PORT/headers"
        ```
        If successful, you get the code `200 OK` in response.

        > [!NOTE]
        > Becuase `httpbin.kyma.example.com` has no DNS record, the command connects directly to the LoadBalancer address and passes the hostname as a `Host` header. Istio uses this header to match the request against the HTTPRoute and forward it to the correct service.

