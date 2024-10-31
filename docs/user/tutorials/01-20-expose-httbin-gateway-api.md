# Expose Workloads Using Gateway API 

Use [Gateway API](https://gateway-api.sigs.k8s.io/) to expose a workload.

> [!WARNING]
> Exposing an unsecured workload to the outside world is a potential security vulnerability, so tread carefully. If you want to use this example in a production environment, make sure to secure your workload.

## Prerequisites

* You have the Istio module added.
* You have a deployed workload. 

## Procedure
1. Export the following values as environment variables:

    ```bash
    export NAMESAPCE={service-namespace}
    export BACKENDNAME={service-name}
    export PORT={service-port}
    ```

    Option | Description |
    ---------|----------|
    NAMESPACE | The name of a namespace you want to use. |
    BACKENDNAME | 	The name of the backend service that you want to use for routing the incoming HTTP traffic. |
    PORT | The port number of the backend server to which requests should be forwarded. |

2. Install Gateway API CustomResourceDefinitions (CRDs) from the standard channel:

    ```bash
    kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null || \
    { kubectl kustomize "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v1.1.0" | kubectl apply -f -; }
    ```

    >[!NOTE]
    > If you’ve already installed Gateway API CRDs from the experimental channel, you must delete them before installing Gateway API CRDs from the standard channel.

3. Create a Kubernetes Gateway to deploy Istio Ingress Gateway:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: gateway
      namespace: ${NAMESPACE}
    spec:
      gatewayClassName: istio
      listeners:
      - name: http
        hostname: "your-domain.kyma.example.com"
        port: 80
        protocol: HTTP
        allowedRoutes:
          namespaces:
            from: Same
    EOF
    ```

    This command deploys the Istio Ingress service in your namespace with the corresponding Kubernetes Service of type LoadBalanced and an assigned external IP address.

4. Create an HTTPRoute to configure access to your workload:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: HTTPRoute
    metadata:
      name: httproute
      namespace: ${NAMESPACE}
    spec:
      parentRefs:
      - name: gateway
      hostnames: ["your-domain.kyma.example.com"]
      rules:
      - matches:
        - path:
            type: PathPrefix
            value: /headers
        backendRefs:
        - name: ${BACKENDNAME}
          namespace: ${NAMESPACE}
          port: ${PORT}
    EOF
    ```

### Results
You've exposed your workload. To access it, follow the steps:

1. Discover Istio Ingress Gateway’s IP and port.
    
    ```bash
    export INGRESS_HOST=$(kubectl get gtw gateway -n $NAMESPACE -o jsonpath='{.status.addresses[0].value}')
    export INGRESS_PORT=$(kubectl get gtw gateway -n $NAMESPACE -o jsonpath='{.spec.listeners[?(@.name=="http")].port}')
    ```

2. Call the service.
    
    ```bash
    curl -s -I -HHost:your-domain.kyma.example.com "http://$INGRESS_HOST:$INGRESS_PORT/headers"
    ```

    >[!NOTE]
    > This task assumes there’s no DNS setup for the `httpbin.kyma.example.com` host, so the call contains the host header.