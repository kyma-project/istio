# Use Gateway API to Expose HTTPBin

This tutorial shows how to expose an HTTPBin Service using Gateway API.

> [!WARNING]
> Exposing a workload to the outside world is a potential security vulnerability, so tread carefully. This example is not meant to be used in production environment. 

## Prerequisites

* Kyma installation with the Istio module enabled. 

## Steps

### Install Gateway API CRDs

Istio module does not install Gateway API CRDs. To install CRDs execute following command:

    ```
    kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null || \
    { kubectl kustomize "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v1.1.0" | kubectl apply -f -; }
    ```

### Create a Workload

1. Export the name of the namespace in which you want to deploy the HTTPBin Service:

    ```
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the HTTPBin Service:

    ```
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

### Expose an HTTPBin Service

1. Create a Kubernetes Gateway to deploy Istio Ingress Gateway.

    ```
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

> [!NOTE]
> This will deploy Istio Ingress service in your namespace with corresponding Kubernetes Service of type `LoadBalanced` and assigned external IP address.

2. Create a HTTPRoute to configure access to your worklad:

    ```
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

### Access an HTTPBin Service

1. Discover Istio Ingress Gateway IP and port

    ```
    export INGRESS_HOST=$(kubectl get gtw httpbin-gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
    export INGRESS_PORT=$(kubectl get gtw httpbin-gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="http")].port}')
    ```

2. Call an HTTPBin Service:

    ```
    curl -s -I -HHost:httpbin.kyma.example.com "http://$INGRESS_HOST:$INGRESS_PORT/headers"
    ```
> [!NOTE]
> This tutorial assumes there's not DNS setup for httpbin.kyma.example.com host, hence setting host header for the call. 

