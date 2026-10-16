# Exposing Workloads Using Gateway API 

The Istio module installs the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs. This tutorial shows how to expose an HTTP workload using a Gateway resource and an HTTPRoute.

## Prerequisites

* You have the Istio module added.
* This tutorial offers two approaches: using the **Kyma domain** (the default domain provided by your Kyma cluster, managed automatically by Gardener) or using a **Custom domain** (your own domain name that you manage through an external DNS provider). For the custom domain approach, you need credentials for a DNS provider supported by Gardener. For the list of supported providers, see [External DNS Management Guidelines](https://github.com/gardener/external-dns-management/blob/master/README.md#external-dns-management).

## Context

In this tutorial, you expose a sample [HTTPBin Service](https://httpbin.org/) outside the cluster using the Kubernetes Gateway API. You can choose between two approaches: using the Kyma domain (managed automatically by Gardener) or using your own custom domain. In both cases, you create a [Gateway](https://gateway-api.sigs.k8s.io/reference/api-types/gateway/) resource that defines the entry point for external traffic, and an [HTTPRoute](https://gateway-api.sigs.k8s.io/reference/api-types/httproute/) that routes incoming traffic to the HTTPBin Service. Istio acts as the Gateway API controller and provisions the infrastructure needed to route traffic. To verify the setup, you call the service and confirm it returns a `200 OK` response.

For details on Gateway API CRD management in the Istio module, see [Gateway API CRDs Management](../00-55-gateway-api-crds.md).

## Procedure

<<!-- tabs:start -->
#### **Custom domain**

1. Export the name of the namespace in which you want to deploy the HTTPBin Service:

    ```bash
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the HTTPBin Service.

    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

    Verify all Pods are running:

    ```bash
    kubectl get pods -l app=httpbin -n $NAMESPACE
    ```

3. Export your domain names.

    ```bash
    export PARENT_DOMAIN={YOUR_PARENT_DOMAIN}
    export SUBDOMAIN=gateway-api-test.${PARENT_DOMAIN}
    export GATEWAY_DOMAIN="*.${SUBDOMAIN}"
    export WORKLOAD_DOMAIN="httpbin.${SUBDOMAIN}"
    echo "Parent domain: $PARENT_DOMAIN"
    echo "Gateway domain: $GATEWAY_DOMAIN"
    echo "Workload domain: $WORKLOAD_DOMAIN"
    ```

4. Create a secret with your DNS provider credentials.

    The `data` field content depends on your DNS provider. To learn how to configure the secret for a specific provider, see [External DNS Management Guidelines](https://github.com/gardener/external-dns-management/blob/master/README.md#external-dns-management). See an example secret for the AWS Route 53 DNS provider. `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are base-64 encoded credentials.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: v1
    kind: Secret
    metadata:
      name: dns-provider-credentials
      namespace: ${NAMESPACE}
    type: Opaque
    data:
      AWS_ACCESS_KEY_ID: {BASE64_AWS_ACCESS_KEY_ID}
      AWS_SECRET_ACCESS_KEY: {BASE64_AWS_SECRET_ACCESS_KEY}
    EOF
    ```

    To verify that the secret is created, run:

    ```bash
    kubectl get secret -n "$NAMESPACE" dns-provider-credentials
    ```

5. Create a `DNSProvider` resource that references the secret with your DNS provider credentials.

    The `spec.type` value depends on your DNS provider. For the list of supported providers and their types, see [External DNS Management Guidelines](https://github.com/gardener/external-dns-management/blob/master/README.md#external-dns-management). See an example `DNSProvider` for the AWS Route 53 DNS provider:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: dns.gardener.cloud/v1alpha1
    kind: DNSProvider
    metadata:
      name: dns-provider
      namespace: ${NAMESPACE}
      annotations:
        dns.gardener.cloud/class: garden
    spec:
      type: aws-route53
      secretRef:
        name: "dns-provider-credentials"
      domains:
        include:
          - "${PARENT_DOMAIN}"
    EOF
    ```

    To verify that the `DNSProvider` is ready, run:

    ```bash
    kubectl get dnsprovider -n "${NAMESPACE}" dns-provider
    ```

6. Create a Kubernetes Gateway resource.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: httpbin-gateway
      namespace: ${NAMESPACE}
      annotations:
        cert.gardener.cloud/purpose: managed
    spec:
      gatewayClassName: istio
      listeners:
      - name: https
        hostname: "${GATEWAY_DOMAIN}"
        port: 443
        protocol: HTTPS
        allowedRoutes:
          namespaces:
            from: Same
        tls:
          certificateRefs:
          - name: httpbin-gatewayapi
    EOF
    ```

    Istio detects the new Gateway resource and automatically provisions a dedicated Envoy proxy pod and a Kubernetes Service of type `LoadBalancer` in your namespace to handle incoming traffic. When you add the `cert.gardener.cloud/purpose: managed` annotation, Gardener Certificate Service requests a TLS certificate from Let's Encrypt and stores it as a secret in the same namespace. For more information, see [Using annotated Gateway API Gateway and/or HTTPRoutes as Source](https://gardener.cloud/docs/extensions/others/gardener-extension-shoot-dns-service/tutorials/gateway-api-gateways/#using-a-gateway-as-a-source).

    Check if the TLS certificate secret was created:

    ```bash
    kubectl get secret httpbin-gatewayapi -n "$NAMESPACE"
    ```

7. Get the external IP address of the Gateway.

    ```bash
    kubectl wait --for=condition=Programmed gateway/httpbin-gateway -n "$NAMESPACE" --timeout=120s
    until LOAD_BALANCER_ADDRESS=$(kubectl get gtw httpbin-gateway -n "$NAMESPACE" -o jsonpath='{.status.addresses[0].value}') && [ -n "$LOAD_BALANCER_ADDRESS" ]; do echo "Waiting for external address..."; sleep 5; done
    echo "Load Balancer address: ${LOAD_BALANCER_ADDRESS}"
    ```

8. Create a `DNSEntry` to point your domain to the Gateway's external IP.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: dns.gardener.cloud/v1alpha1
    kind: DNSEntry
    metadata:
      name: dns-entry
      namespace: ${NAMESPACE}
      annotations:
        dns.gardener.cloud/class: garden
    spec:
      dnsName: "${GATEWAY_DOMAIN}"
      ttl: 600
      targets:
        - "${LOAD_BALANCER_ADDRESS}"
    EOF
    ```

    Check if the `DNSEntry` is ready:

    ```bash
    kubectl get dnsentry -n "${NAMESPACE}" dns-entry
    ```

9. Create an HTTPRoute to configure access to your workload:

    The HTTPRoute exposes only the `/headers` endpoint of the HTTPBin Service, which returns the headers of the incoming request. The HTTPBin Service listens on port `8000`.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: gateway.networking.k8s.io/v1
    kind: HTTPRoute
    metadata:
      name: httpbin
      namespace: ${NAMESPACE}
    spec:
      parentRefs:
      - name: httpbin-gateway
      hostnames: ["${WORKLOAD_DOMAIN}"]
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

10. Verify access to the HTTPBin Service.

    ```bash
    curl "https://$WORKLOAD_DOMAIN/headers"
    ```

    After DNS propagates, it may take a few minutes for your system's DNS cache to refresh. When successful, you get the code `200 OK` in response.

#### **Kyma domain**

1. Export the name of the namespace in which you want to deploy the HTTPBin Service:
    ```bash
    export NAMESPACE={NAMESPACE_NAME}
    ```

2. Create a namespace with Istio injection enabled and deploy the HTTPBin Service.

    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    kubectl create -n $NAMESPACE -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

    Verify all Pods are running:

    ```bash
    kubectl get pods -l app=httpbin -n $NAMESPACE
    ```

3. Export your domain name.

    ```bash
    export PARENT_DOMAIN=$(kubectl get configmap -n kube-system shoot-info -o jsonpath="{.data.domain}")
    export GATEWAY_DOMAIN="gateway-api-test.${PARENT_DOMAIN}"
    export WORKLOAD_DOMAIN="httpbin.${GATEWAY_DOMAIN}"
    echo "Parent domain: $PARENT_DOMAIN"
    echo "Gateway domain: $GATEWAY_DOMAIN"
    echo "Workload domain: $WORKLOAD_DOMAIN"
    ```

4. Create a Kubernetes Gateway resource.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: httpbin-gateway
      namespace: ${NAMESPACE}
      annotations:
        cert.gardener.cloud/purpose: managed
        dns.gardener.cloud/dnsnames: "*.${GATEWAY_DOMAIN}"
        dns.gardener.cloud/class: garden
        dns.gardener.cloud/ttl: "600"
    spec:
      gatewayClassName: istio
      listeners:
      - name: https
        hostname: "*.${GATEWAY_DOMAIN}"
        port: 443
        protocol: HTTPS
        allowedRoutes:
          namespaces:
            from: Same
        tls:
          certificateRefs:
          - name: httpbin-gatewayapi
    EOF
    ```

    Istio detects the new Gateway resource and automatically provisions a dedicated Envoy proxy pod and a Kubernetes Service of type `LoadBalancer` in your namespace to handle incoming traffic. At the same time, two Gardener controllers react to the annotations: the DNS controller creates a DNS record pointing to the Gateway's external IP, and the Certificate Service requests a TLS certificate from Let's Encrypt and stores it as a secret in the same namespace. For more information, see [Using annotated Gateway API Gateway and/or HTTPRoutes as Source](https://gardener.cloud/docs/extensions/others/gardener-extension-shoot-dns-service/tutorials/gateway-api-gateways/#using-a-gateway-as-a-source) and [External DNS Management](https://gardener.cloud/docs/getting-started/features/dns-management/).

    Check if the TLS certificate secret was created:

    ```bash
    kubectl get secret httpbin-gatewayapi -n "$NAMESPACE"
    ```

    Check if DNS has propagated:

    ```bash
    dig +short "$WORKLOAD_DOMAIN"
    ```

5. Create an HTTPRoute to configure access to your workload:

    The HTTPRoute exposes only the `/headers` endpoint of the HTTPBin Service, which returns the headers of the incoming request. The HTTPBin Service listens on port `8000`.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: gateway.networking.k8s.io/v1
    kind: HTTPRoute
    metadata:
      name: httpbin
      namespace: ${NAMESPACE}
    spec:
      parentRefs:
      - name: httpbin-gateway
      hostnames: ["${WORKLOAD_DOMAIN}"]
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

6. Verify access to the HTTPBin Service.

    ```bash
    curl -I "https://$WORKLOAD_DOMAIN/headers"
    ```

    After DNS propagates, it may take a few minutes for your system's DNS cache to refresh. When successful, you get the code `200 OK` in response.

<!-- tabs:end -->