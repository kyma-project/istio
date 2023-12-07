# Forward Client IP in X-Forwarded-For header

Many applications need to know the client IP address of an originating request to behave properly. Usual use-cases include workloads that require the 
client IP address to restrict their access. The ability to provide client attributes to services has long been a staple of reverse proxies. 
To forward client attributes to destination workloads, proxies use the X-Forwarded-For (XFF) header. For more information on XFF, see 
the [IETF’s RFC documentation](https://datatracker.ietf.org/doc/html/rfc7239) and [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#x-forwarded-for).

> **NOTE:** The X-Forwarded-For header is only supported on AWS clusters.

## Prerequisites

* An AWS cluster
* [Istio module latest release version installed](../../../README.md#install-kyma-istio-operator-and-istio-from-the-latest-release)

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
        kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"numTrustedProxies":2}}}'
        ```

   #### **Kyma Dashboard**
    1. Navigate to the `Cluster Details`
    2. Open the `istio` module configuration
    3. Click `Edit`
    4. Add the numTrustedProxies:
    ![Add the numTrustedProxies](./assets/01-00-num-trusted-proxies-ui.svg)
    <!-- tabs:end -->
