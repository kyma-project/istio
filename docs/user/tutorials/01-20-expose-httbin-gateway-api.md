# Exposing Workloads Using Gateway API 

The Istio module automatically installs the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs. This tutorial shows how to expose an HTTP workload using an HTTPRoute and Istio Ingress Gateway. For details on CRD management, see [Gateway API CRDs Management](../00-55-gateway-api-crds.md).

## Prerequisites

* You have the Istio module added.

## Procedure

1. Export the name of the namespace in which you want to deploy a sample HTTPBin Service:
    ```bash
    export NAMESPACE={service-namespace}
    ```

2. Create a namespace with Istio injection enabled and deploy the HTTPBin Service:
    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

3. To expose a workload, create a Kubernetes Gateway to deploy Istio Ingress Gateway:

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

    This command deploys the Istio Ingress service in your namespace with the corresponding Kubernetes Service of type LoadBalancer and an assigned external IP address.

4. Create an HTTPRoute to configure access to your workload:

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

5. To access your exposed workload, follow the steps:

    1. Discover Istio Ingress Gateway’s IP and port.
        
        ```bash
        export INGRESS_HOST=$(kubectl get gtw httpbin-gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
        export INGRESS_PORT=$(kubectl get gtw httpbin-gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="http")].port}')
        ```

    2. Call the service.
        
        ```bash
        curl -s -I -HHost:httpbin.kyma.example.com "http://$INGRESS_HOST:$INGRESS_PORT/headers"
        ```
        If successful, you get the code `200 OK` in response.

        >[!NOTE]
        > This task assumes there’s no DNS setup for the `httpbin.kyma.example.com` host, so the call contains the host header.