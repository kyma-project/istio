# Exposing Workloads Using Gateway API 

The Istio module installs the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs. This tutorial shows how to expose an HTTP workload using a Gateway resource and an HTTPRoute.

## Prerequisites

* You have the Istio module added.
* You can use a Kyma domain (the default domain provided by your Kyma cluster, managed automatically by Gardener) or a custom domain (your own domain name that you manage through an external DNS provider). For the custom domain approach, you need credentials for a DNS provider supported by Gardener. For the list of supported providers, see [External DNS Management Guidelines](https://github.com/gardener/external-dns-management/blob/master/README.md#external-dns-management).

## Context

In this tutorial, you expose a sample [HTTPBin Service](https://httpbin.org/) outside the cluster using the Kubernetes Gateway API. You can choose between two approaches: using the Kyma domain (managed automatically by Gardener) or using your own custom domain. In both cases, you create a [Gateway](https://gateway-api.sigs.k8s.io/reference/api-types/gateway/) resource that defines the entry point for external traffic, and an [HTTPRoute](https://gateway-api.sigs.k8s.io/reference/api-types/httproute/) that routes incoming traffic to the HTTPBin Service. Istio acts as the Gateway API controller and provisions the infrastructure needed to route traffic. To verify the setup, you call the service and confirm it returns a `200 OK` response.

For details on Gateway API CRD management in the Istio module, see [Gateway API CRDs Management](../00-55-gateway-api-crds.md).

## Procedure

<!-- tabs:start -->
#### **Custom domain**

1. Create a test namespace with Istio injection enabled and deploy the HTTPBin Service.

    ```bash
    kubectl create ns test
    kubectl label namespace test istio-injection=enabled --overwrite
    kubectl create -n test -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

    Verify all Pods are running:

    ```bash
    kubectl get pods -l app=httpbin -n test
    ```

2. Export your domain names.

    ```bash
    export PARENT_DOMAIN={YOUR_PARENT_DOMAIN}
    export SUBDOMAIN=tls.${PARENT_DOMAIN}
    export GATEWAY_DOMAIN="*.${SUBDOMAIN}"
    export WORKLOAD_DOMAIN="httpbin.${SUBDOMAIN}"
    echo "Parent domain: $PARENT_DOMAIN"
    echo "Subdomain: $SUBDOMAIN"
    echo "Gateway domain: $GATEWAY_DOMAIN"
    echo "Workload domain: $WORKLOAD_DOMAIN"
    ```
    | Placeholder | Example domain name | Description |
    |---------|----------|---------|
    | **PARENT_DOMAIN** | `my-own-domain.example.com` | The domain name available in the public DNS zone. |
    | **SUBDOMAIN** | `tls.my-own-domain.example.com` | A subdomain created under the parent domain, specifically for the TLS Gateway. |
    | **GATEWAY_DOMAIN** | `*.tls.my-own-domain.example.com` | A wildcard domain covering all possible first-level subdomains under the TLS subdomain. When configuring the Gateway, this allows you to expose workloads on multiple hosts (for example, `httpbin.tls.my-default-domain.kyma.ondemand.com`, `test.tls.my-default-domain.kyma.ondemand.com`) without creating separate Gateway rules for each one.|
    | **WORKLOAD_DOMAIN** | `httpbin.tls.my-own-domain.example.com` | The specific domain assigned to your workload. |

3. Create a secret with your DNS provider credentials.

    The information you provide to the **data** field differs depending on the DNS provider you're using. The DNS provider must be supported by Gardener. To learn how to configure the Secret for a specific provider, follow [External DNS Management Guidelines](https://github.com/gardener/external-dns-management/blob/master/README.md#external-dns-management).
    
    See an example Secret for the AWS Route 53 DNS provider. **AWS_ACCESS_KEY_ID** and **AWS_SECRET_ACCESS_KEY** are base-64 encoded credentials.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: v1
    kind: Secret
    metadata:
      name: dns-provider-credentials
      namespace: test
    type: Opaque
    data:
      AWS_ACCESS_KEY_ID: {BASE64_AWS_ACCESS_KEY_ID}
      AWS_SECRET_ACCESS_KEY: {BASE64_AWS_SECRET_ACCESS_KEY}
    EOF
    ```

    To verify that the secret is created, run:

    ```bash
    kubectl get secret -n test dns-provider-credentials
    ```

4. Create a DNSProvider that references the secret with your DNS provider credentials.

    The `spec.type` value depends on your DNS provider. For the list of supported providers and their types, see [External DNS Management Guidelines](https://github.com/gardener/external-dns-management/blob/master/README.md#external-dns-management). 
    
    See an example `DNSProvider` for the AWS Route 53 DNS provider:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: dns.gardener.cloud/v1alpha1
    kind: DNSProvider
    metadata:
      name: dns-provider
      namespace: test
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
    kubectl get dnsprovider -n test dns-provider
    ```

5. Create the server's certificate for the specified subdomain.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: cert.gardener.cloud/v1alpha1
    kind: Certificate
    metadata:
      name: domain-certificate
      namespace: "test"
    spec:
      secretName: httpbin-gatewayapi
      commonName: "${GATEWAY_DOMAIN}"
      issuerRef:
        name: garden
    EOF
    ```

6. Create a Kubernetes Gateway resource.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: httpbin-gateway
      namespace: test
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

    Istio detects the new Gateway resource and automatically provisions a dedicated Envoy proxy pod and a Kubernetes Service of type `LoadBalancer` in your namespace to handle incoming traffic.

7. Get the external IP address of the Gateway.

    ```bash
    LOAD_BALANCER_ADDRESS=$(kubectl get gtw httpbin-gateway -n test -o jsonpath='{.status.addresses[0].value}')
    echo "Load Balancer address: ${LOAD_BALANCER_ADDRESS}"
    ```

8. Create a DNSEntry to point your domain to the Gateway's external IP.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: dns.gardener.cloud/v1alpha1
    kind: DNSEntry
    metadata:
      name: dns-entry
      namespace: test
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
    kubectl get dnsentry -n test dns-entry
    ```

9. Create an HTTPRoute to configure access to your workload:

    The HTTPRoute exposes only the `/headers` endpoint of the HTTPBin Service, which returns the headers of the incoming request. The HTTPBin Service listens on port `8000`.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: HTTPRoute
    metadata:
      name: httpbin
      namespace: test
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
          namespace: test
          port: 8000
    EOF
    ```

    Check that the route is valid and all references are resolved:

    ```bash
    kubectl describe httproute httpbin -n test
    ```

10. Verify access to the HTTPBin Service.

    ```bash
    curl "https://$WORKLOAD_DOMAIN/headers"
    ```

    After DNS propagates, it may take a few minutes for your system's DNS cache to refresh. When successful, you get the code `200 OK` in response.

#### **Kyma domain**

1. Create a namespace with Istio injection enabled and deploy the HTTPBin Service.

    ```bash
    kubectl create ns test
    kubectl label namespace test istio-injection=enabled --overwrite
    kubectl create -n test -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
    ```

    Verify all Pods are running:

    ```bash
    kubectl get pods -l app=httpbin -n test
    ```

2. Export your domain name.

    ```bash
    export PARENT_DOMAIN=$(kubectl get configmap -n kube-system shoot-info -o jsonpath="{.data.domain}")
    export SUBDOMAIN="tls.${PARENT_DOMAIN}"
    export GATEWAY_DOMAIN="*.${SUBDOMAIN}"
    export WORKLOAD_DOMAIN="httpbin.${SUBDOMAIN}"
    echo "Parent domain: $PARENT_DOMAIN"
    echo "Subdomain: $SUBDOMAIN"
    echo "Gateway domain: $GATEWAY_DOMAIN"
    echo "Workload domain: $WORKLOAD_DOMAIN"
    ```

    | Placeholder         | Example domain name                                | Description                                                                                                                                                                                                                                                                                                                                       |
    |---------------------|----------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
    | **PARENT_DOMAIN**   | `my-default-domain.kyma.ondemand.com`              | The default domain of your Kyma cluster retrieved from the default TLS Gateway `kyma-gateway`.                                                                                                                                                                                                                                                    |
    | **SUBDOMAIN**       | `tls.my-default-domain.kyma.ondemand.com`         | A subdomain created under the parent domain, specifically for the TLS Gateway. Having a separate subdomain is required if you use the default domain of your Kyma cluster, as the parent domain name is already assigned to the TLS Gateway `kyma-gateway` installed in your cluster by default.                                                 |
    | **GATEWAY_DOMAIN**  | `*.tls.my-default-domain.kyma.ondemand.com`       | A wildcard domain covering all possible first-level subdomains under the TLS subdomain. When configuring the Gateway, this allows you to expose workloads on multiple hosts (for example, `httpbin.tls.my-default-domain.kyma.ondemand.com`, `test.tls.my-default-domain.kyma.ondemand.com`) without creating separate Gateway rules for each one. |
    | **WORKLOAD_DOMAIN** | `httpbin.tls.my-default-domain.kyma.ondemand.com` | The specific domain assigned to your sample workload (HTTPBin Service) in this tutorial.                                                                                                                                                                                                                                                          |

3. Create the server's certificate for the specified subdomain.
    
    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: cert.gardener.cloud/v1alpha1
    kind: Certificate
    metadata:
      name: domain-certificate
      namespace: "test"
    spec:
      secretName: httpbin-gatewayapi
      commonName: "${GATEWAY_DOMAIN}"
      issuerRef:
        name: garden
    EOF
    ```

4. Create a Kubernetes Gateway resource.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: httpbin-gateway
      namespace: test
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

    Istio detects the new Gateway resource and automatically provisions a dedicated Envoy proxy pod and a Kubernetes Service of type `LoadBalancer` in your namespace to handle incoming traffic.

5. Get the external IP address of the Gateway.

    ```bash
    export LOAD_BALANCER_ADDRESS=$(kubectl get gtw httpbin-gateway -n test -o jsonpath='{.status.addresses[0].value}')
    echo "Load Balancer address: ${LOAD_BALANCER_ADDRESS}"
    ```

6. Create a DNSEntry to point your domain to the Gateway's external IP.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: dns.gardener.cloud/v1alpha1
    kind: DNSEntry
    metadata:
      name: dns-entry
      namespace: test
      annotations:
        dns.gardener.cloud/class: garden
    spec:
      dnsName: "${GATEWAY_DOMAIN}"
      ttl: 600
      targets:
        - "${LOAD_BALANCER_ADDRESS}"
    EOF
    ```

7. Create an HTTPRoute to configure access to your workload:

    The HTTPRoute exposes only the `/headers` endpoint of the HTTPBin Service, which returns the headers of the incoming request. The HTTPBin Service listens on port `8000`.

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.networking.k8s.io/v1
    kind: HTTPRoute
    metadata:
      name: httpbin
      namespace: test
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
          namespace: test
          port: 8000
    EOF
    ```

    Check that the route is valid and all references are resolved:

    ```bash
    kubectl describe httproute httpbin -n test
    ```

8. Verify access to the HTTPBin Service.

    ```bash
    curl -I "https://$WORKLOAD_DOMAIN/headers"
    ```

    After DNS propagates, it may take a few minutes for your system's DNS cache to refresh. When successful, you get the code `200 OK` in response.

<!-- tabs:end -->