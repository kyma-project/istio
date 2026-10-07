# Exposing a TCP Service Using Gateway API

The Istio module automatically installs the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs. This tutorial shows how to expose a TCP service using a TCPRoute and Istio Ingress Gateway. For details on CRD management, see [Gateway API CRDs Management](../00-55-gateway-api-crds.md).

## Prerequisites

* You have the Istio module added.

## Procedure

1. Export the name of the namespace in which you want to deploy the TCPEcho Service:

    ```bash
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the TCPEcho Service:

    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.31/samples/tcp-echo/tcp-echo.yaml
    ```

3. To expose a TCPEcho Service, create a Kubernetes Gateway to deploy Istio Ingress Gateway:

    ```bash
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
    > This command deploys the Istio Ingress service in your namespace with the corresponding Kubernetes Service of type `LoadBalanced` and an assigned external IP address.

4. Create a TCPRoute to configure access to your workload:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
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

5. Send TCP Traffic to a TCPEcho Service

    1. Discover Istio Ingress Gateway's IP and port:

        ```bash
        export INGRESS_HOST=$(kubectl get gtw tcp-echo-gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
        export INGRESS_PORT=$(kubectl get gtw tcp-echo-gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="tcp-31400")].port}')
        ```

    2. Deploy a `sleep` Service:

        ```bash
        kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.31/samples/sleep/sleep.yaml
        ```

    3. Send TCP traffic:

        ```bash
        export SLEEP=$(kubectl get pod -l app=sleep -n $NAMESPACE -o jsonpath={.items..metadata.name})
        for i in {1..3}; do \
        kubectl exec "$SLEEP" -c sleep -n $NAMESPACE -- sh -c "(date; sleep 1) | nc $INGRESS_HOST $INGRESS_PORT"; \
        done
        ```
        You should see similar output:
        ```
        hello Mon Jul 29 12:43:56 UTC 2024
        ```