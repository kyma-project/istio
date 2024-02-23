# External authorization provider

This tutorial shows how to expose and secure an HTTPBin service using an external authorization provider. This tutorial assumes usage of the Kyma domain.

## Prerequisites

* Kyma installation with the `api-gateway` and `istio` components enabled.
* Deploy [a sample HTTPBin Service](../01-00-create-workload.md).

## Steps

### Expose httpbin

1. Set your kyma domain address as an environment variable:
    ```
    export DOMAIN_TO_EXPOSE_WORKLOADS={KYMA_DOMAIN_NAME}
    ```

2. Expose an instance of the HTTPBin Service by creating APIRule CR in your namespace.
    ```
    cat <<EOF | kubectl apply -f -
    apiVersion: gateway.kyma-project.io/v1beta1
    kind: APIRule
    metadata:
      name: httpbin
      namespace: $NAMESPACE
    spec:
      host: httpbin.$DOMAIN_TO_EXPOSE_WORKLOADS
      service:
        name: $SERVICE_NAME
        namespace: $NAMESPACE
        port: 8000
      gateway: kyma-system/kyma-gateway
      rules:
        - path: /.*
          methods: ["GET"]
          accessStrategies:
            - handler: no_auth
    EOF
    ```

### Install and configure oauth2-proxy
- Manual installation of oauth2-proxy, for example [using Helm](https://github.com/oauth2-proxy/manifests/tree/main/helm/oauth2-proxy).

1. Set your configuration values as environment variables:
    ```
    export CLIENT_ID={CLIENT_ID}
    export CLIENT_SECRET={CLIENT_SECRET}
    export COOKIE_SECRET={COOKIE_SECRET}
    export OIDC_ISSUER_URL={OIDC_ISSUER_URL}
    ```

2. Use the following command to create a `values.yaml` file with the configuration of oauth2-proxy:
    ```
    cat <<EOF > values.yaml
    config:
      clientID: $CLIENT_ID
      clientSecret: $CLIENT_SECRET
      cookieSecret: $COOKIE_SECRET
    extraArgs:
      provider: oidc
      cookie-secure: false
      cookie-domain: "$DOMAIN_TO_EXPOSE_WORKLOADS"
      cookie-samesite: lax
      set-xauthrequest: true
      whitelist-domain: "*.$DOMAIN_TO_EXPOSE_WORKLOADS:*"
      reverse-proxy: true
      pass-access-token: true 
      set-authorization-header: true
      pass-authorization-header: true
      scope: "openid email"
      upstream: static://200
      skip-provider-button: true
      redirect-url: "https://oauth2-proxy.$DOMAIN_TO_EXPOSE_WORKLOADS/oauth2/callback"
      oidc-issuer-url: https://$OIDC_ISSUER_URL
      code-challenge-method: S256
    EOF
    ```

3. Install oauth2-proxy with your configuration using Helm:
    ```
    helm install custom oauth2-proxy/oauth2-proxy -f values.yaml
    ```

### Configure Istio CR with an external authorization provider

1. Here is an example of Istio CR configuration with an external authorization provider:
    ```
    cat <<EOF | kubectl apply -f -
    apiVersion: operator.kyma-project.io/v1alpha1
    kind: Istio
    metadata:
      name: default
      namespace: kyma-system
    spec:
      config:
        numTrustedProxies: 1
        authorizers:
        - name: "oauth2-proxy"
          service: "custom-oauth2-proxy.default.svc.cluster.local"
          port: 80
          headers:
            inCheck:
              include: ["authorization", "cookie"]
              add:
                x-auth-request-redirect: "https://%REQ(:authority)%%REQ(x-envoy-original-path?:path)%"
            toUpstream:
              onAllow: ["authorization", "path", "x-auth-request-user", "x-auth-request-email", "x-auth-request-access-token"]
            toDownstream:
              onAllow: ["set-cookie"]
              onDeny: ["content-type", "set-cookie"]
    EOF
    ```

2. Additionally, you have to create an AuthorizationPolicy CR with CUSTOM action and oauth2-proxy provider:
    ```
    cat <<EOF | kubectl apply -f -
    apiVersion: security.istio.io/v1
    kind: AuthorizationPolicy
    metadata:
        name: ext-authz
        namespace: test
    spec:
      action: CUSTOM
      provider:
        name: oauth2-proxy
      rules:
      - to:
        - operation:
            paths:
            - /headers
      selector:
        matchLabels:
          app: httpbin
    EOF
    ```

3. Finally, create a DestinationRule with a traffic policy for the external authorization provider:
    ```
    cat <<EOF | kubectl apply -f -
    apiVersion: networking.istio.io/v1beta1
    kind: DestinationRule
    metadata:
      name: external-authz-https
      namespace: istio-system
    spec:
      host: $OIDC_ISSUER_URL
      trafficPolicy:
        tls:
          mode: SIMPLE
    EOF
    ```
