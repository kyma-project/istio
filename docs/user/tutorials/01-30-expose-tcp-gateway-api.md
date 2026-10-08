# Exposing a TCP Service Using Gateway API

The Istio module installs the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs. This tutorial shows how to expose a TCP service using a Gateway resource and a TCPRoute.

## Prerequisites

* You have the Istio module added.

## Context

In this tutorial, you expose a sample TCP Echo Service outside the cluster using the Kubernetes Gateway API. First, you create a [Gateway](https://gateway-api.sigs.k8s.io/reference/api-types/gateway/) resource that defines the entry point for external traffic. Then, you create a [TCPRoute](https://gateway-api.sigs.k8s.io/reference/api-types/tcproute/) that defines how incoming traffic is forwarded to the TCP Echo Service. Istio acts as the Gateway API controller and provisions the infrastructure needed to route traffic. To verify the setup, you send TCP traffic through the gateway and confirm that the service echoes it back.

For details on Gateway API CRD management in the Istio module, see [Gateway API CRDs Management](../00-55-gateway-api-crds.md).

## Procedure

1. Export the name of the namespace in which you want to deploy the TCP Echo Service:

    ```bash
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the TCP Echo Service.

    TCP Echo is a sample service that echoes back any TCP traffic it receives, prefixed with `hello`. It is used to verify that traffic flows correctly through the gateway.

    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.31/samples/tcp-echo/tcp-echo.yaml
    ```

    Verify all Pods are ready:

    ```bash
    kubectl get pods -l app=tcp-echo -n $NAMESPACE
    ```

3. To create an entry point for external TCP traffic, create a Kubernetes Gateway resource:

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

    Istio detects the new Gateway resource and automatically provisions a dedicated Envoy proxy pod and a Kubernetes Service of type `LoadBalancer` in your namespace to handle incoming traffic.

    Verify that the Gateway is programmed and has an external address assigned — this confirms the LoadBalancer is ready to accept traffic:

    ```bash
    kubectl get gtw tcp-echo-gateway -n $NAMESPACE
    ```
    
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

    Verify that the route is valid and all references are resolved:

    ```bash
    kubectl describe tcproute tcp-echo -n $NAMESPACE
    ```

5. Verify access to the TCP Echo Service:

    1. Discover the gateway's external IP and port:

        ```bash
        export INGRESS_HOST=$(kubectl get gtw tcp-echo-gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
        export INGRESS_PORT=$(kubectl get gtw tcp-echo-gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="tcp-31400")].port}')
        until export INGRESS_IP=$(dig +short $INGRESS_HOST A | head -1); [ -n "$INGRESS_IP" ]; do echo "Waiting..."; sleep 5; done
        echo "Ingress IP: $INGRESS_IP, port: $INGRESS_PORT"
        ```

        On AWS, the load balancer is assigned a hostname rather than a direct IP. Because `nc` requires a numeric IP, the commands resolve the hostname to an IP using `dig`.

    2. Deploy a `sleep` Pod, which acts as a TCP client inside the mesh and sends traffic to the TCP Echo Service through the gateway.

        ```bash
        kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/release-1.31/samples/sleep/sleep.yaml
        ```

        Verify the Pod is running:

        ```bash
        kubectl get pods -l app=sleep -n $NAMESPACE
        ```

    3. Send TCP traffic:

        ```bash
        export SLEEP=$(kubectl get pod -l app=sleep -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
        for i in {1..3}; do \
        kubectl exec "$SLEEP" -c sleep -n $NAMESPACE -- sh -c "(date; sleep 1) | nc $INGRESS_IP $INGRESS_PORT"; \
        done
        ```
        
        See an example output:
        
        ```bash
        hello Thu Oct  8 09:45:36 UTC 2026
        hello Thu Oct  8 09:45:38 UTC 2026
        hello Thu Oct  8 09:45:39 UTC 2026
        ```