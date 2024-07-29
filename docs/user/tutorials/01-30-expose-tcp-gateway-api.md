# Use Gateway API to Expose HTTPBin

This tutorial shows how to expose an HTTPBin Service using Gateway API.

> [!WARNING]
> This tutorial is based on experimental Istio module.
> Exposing a workload to the outside world is a potential security vulnerability, so tread carefully. This example is not meant to be used in production environment. 

## Prerequisites

* Experimental Istio module installation

## Steps

### Configure Gateway API alpha support

Edit Istio CR by setting `enableAlphaGatewayAPI` to true. Your Istio CR should look:

    ```
    ...
    spec:
      config: {}
      experimental:
        pilot:
          enableAlphaGatewayAPI: true
          enableMultiNetworkDiscoverGatewayAPI: false
    ...
    ```

### Install experimental version of Gateway API CRDs

Istio module does not install Gateway API CRDs. To install CRDs execute following command:

    ```
    kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null || \
    { kubectl kustomize "github.com/kubernetes-sigs/gateway-api/config/crd/experimental?ref=v1.1.0" | kubectl apply -f -; }
    ```

### Create a Workload

1. Export the name of the namespace in which you want to deploy the TCPEcho Service:

    ```
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the TCPEcho Service:

    ```
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.22/samples/tcp-echo/tcp-echo.yaml
    ```

### Expose an TCPEcho Service

1. Create a Kubernetes Gateway to deploy Istio Ingress Gateway.

    ```
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: tcp-echo-gateway
      namespace: ${NAMESPACE}
    spec:
      gatewayClassName: istio
      listeners:
      - name: tcp-31400
        port: 31400
        protocol: TCP
        allowedRoutes:
          namespaces:
            from: Same
    EOF
    ```

> [!NOTE]
> This will deploy Istio Ingress service in your namespace with corresponding Kubernetes Service of type `LoadBalanced` and assigned external IP address.

2. Create a TCPRoute to configure access to your worklad:

    ```
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1alpha2
    kind: TCPRoute
    metadata:
      name: tcp-echo
      namespace: ${NAMESPACE}
    spec:
      parentRefs:
      - name: tcp-echo-gateway
        sectionName: tcp-31400
      rules:
      - backendRefs:
        - name: tcp-echo
          port: 9000
    EOF
    ```

### Send TCP traffic to an TCPEcho Service

1. Discover Istio Ingress Gateway IP and port

    ```
    export INGRESS_HOST=$(kubectl get gtw tcp-echo-gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
    export INGRESS_PORT=$(kubectl get gtw tcp-echo-gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="tcp-31400")].port}')
    ```

2. Deploy Sleep service:

    ```
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.22/samples/sleep/sleep.yaml
    ```


2. Send TCP traffic:

    ```
    export SLEEP=$(kubectl get pod -l app=sleep -n $NAMESPACE -o jsonpath={.items..metadata.name})
    for i in {1..3}; do \
    kubectl exec "$SLEEP" -c sleep -n $NAMESPACE -- sh -c "(date; sleep 1) | nc $INGRESS_HOST $INGRESS_PORT"; \
    done
    ```
    You should see simmilar output:
    ```
    hello Mon Jul 29 12:43:56 UTC 2024
    ```

