# Forward Client IP in X-Forwarded-For header

Many applications need to know the client IP address of an originating request to behave properly. Usual use-cases include workloads that require the 
client IP address to restrict their access. The ability to provide client attributes to services has long been a staple of reverse proxies. 
To forward client attributes to destination workloads, proxies use the X-Forwarded-For (XFF) header. For more information on XFF, see 
the [IETF’s RFC documentation](https://datatracker.ietf.org/doc/html/rfc7239) and [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#x-forwarded-for).

> **NOTE:** The X-Forwarded-For header is only supported on AWS clusters.

## Prerequisites

* An AWS cluster
* An enabled Istio module or an installed [Kyma Istio Operator](../../../README.md#install-kyma-istio-operator-and-istio-from-the-latest-release)
* An enabled API Gateway module or an installed [Kyma API Gateway Operator](https://github.com/kyma-project/api-gateway#installation). Alternatively, a [custom Istio Gateway](https://kyma-project.io/#/api-gateway/user/tutorials/01-20-set-up-tls-gateway) can be used.

## Steps

### Configure the number of trusted proxies in the Istio CR

Applications rely on reverse proxies to forward the client IP address in a request using the XFF header. However, due to 
the variety of network topologies, configuration property numTrustedProxies must be specified with the number of trusted proxies deployed 
in front of the Istio Gateway proxy, so that the client address can be extracted correctly.

1. Add the numTrustedProxies to Istio CR:

<!-- tabs:start -->
   #### **kubectl**
   1. Add the numTrustedProxies to Istio CR:
      ```bash
      kubectl patch istios/default -n kyma-system --type merge -p '{"spec":{"config":{"numTrustedProxies": 1}}}'
      ```

   #### **Kyma Dashboard**
   1. Navigate to the `Cluster Details`
   2. Open the `istio` module configuration
   3. Click `Edit`
   4. Add the numTrustedProxies:  
      ![Add the numTrustedProxies](./assets/01-00-num-trusted-proxies-ui.svg)
<!-- tabs:end -->


### Create a workload for verification

1. Create a HttpBin workload as described in the [Create a workload tutorial](https://kyma-project.io/#/api-gateway/user/tutorials/01-00-create-workload).
2. Expose the HttpBin workload using a Virtual Service.
    ```bash
    export DOMAIN_TO_EXPOSE_WORKLOADS={KYMA_DOMAIN_NAME}
    export GATEWAY=kyma-system/kyma-gateway

    cat <<EOF | kubectl apply -f -
    apiVersion: networking.istio.io/v1alpha3
    kind: VirtualService
    metadata:
      name: httpbin
      namespace: $NAMESPACE
    spec:
      hosts:
      - "httpbin.$DOMAIN_TO_EXPOSE_WORKLOADS"
      gateways:
      - $GATEWAY
      http:
      - match:
        - uri:
            prefix: /
        route:
        - destination:
            port:
              number: 8000
            host: httpbin.$NAMESPACE.svc.cluster.local
    EOF
    ```

### Verify the X-Forwarded-For and X-Envoy-External-Address header
1. Get you public IP address, for example by using the following command:
    ```bash
    curl -s https://api.ipify.org
    ```

2. Send a request to the HttpBin workload using the following command:
    ```bash
    curl -s "https://httpbin.$DOMAIN_TO_EXPOSE_WORKLOADS/get?show_env=true"
    ```
3. Verify that the response contains the `X-Forwarded-For` and `X-Envoy-External-Address` header with your public IP address, for example:
    ```json
    {
      "args": {
        "show_env": "true"
      },
      "headers": {
        "Accept": ...,
        "Host": ...,
        "User-Agent": ...,
        "X-Envoy-Attempt-Count": ...,
        "X-Envoy-External-Address": "165.1.187.197",
        "X-Forwarded-Client-Cert": ...,
        "X-Forwarded-For": "165.1.187.197",
        "X-Forwarded-Proto": ...,
        "X-Request-Id": ...
      },
      "origin": "165.1.187.197",
      "url": ...
    }
    ``` 
