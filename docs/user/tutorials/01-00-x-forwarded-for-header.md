# Forward a Client IP in the X-Forwarded-For Header

Many applications need to know the client IP address of an originating request to behave properly. Usual use cases include workloads that require the client IP address to restrict their access. The ability to provide client attributes to services has long been a staple of reverse proxies, which use the X-Forwarded-For (XFF) header to forward client attributes to destination workloads. For more information on XFF, see 
the [IETF’s RFC documentation](https://datatracker.ietf.org/doc/html/rfc7239) and [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#x-forwarded-for).

## Prerequisites

* The Istio module enabled or [Kyma Istio Operator](../../../README.md#install-kyma-istio-operator-and-istio-from-the-latest-release) installed
* [Istio Gateway](https://kyma-project.io/#/api-gateway/user/tutorials/01-20-set-up-tls-gateway) set up

## Steps

### Configure the Number of Trusted Proxies in the Istio Custom Resource

Applications rely on reverse proxies to forward the client IP address in a request using the XFF header. However, due to 
the variety of network topologies, you must specify the configuration property **numTrustedProxies**, so that the client address can be extracted correctly. This property indicates the number of trusted proxies deployed
in front of the Istio Gateway proxy.

Add **numTrustedProxies** to the Istio custom resource:

<!-- tabs:start -->
#### **kubectl**
Run the following command:

  ```bash
  kubectl patch istios/default -n kyma-system --type merge -p '{"spec":{"config":{"numTrustedProxies": 1}}}'
  ```

#### **Kyma Dashboard**
1. Navigate to **Cluster Details** and select **Modify Modules**.
2. Choose the Istio module and select **Edit**.
3. In the `General` section, add the number of trusted proxies.
4. Select **Save**.
<!-- tabs:end -->

### Configure Gateway External Traffic Policy in the Istio Custom Resource (GCP and Azure only)

If you are using a GCP or Azure cluster, you must also set the **gatewayExternalTrafficPolicy** to `Local` in order to include the client's IP address in the XFF header. Skip this step if you're using a different cloud service provider.

For production Deployments, it is strongly recommended to deploy an Ingress Gateway Pod to multiple nodes if you enable `externalTrafficPolicy: Local`. Otherwise, this creates a situation where only nodes with an active Ingress Gateway Pod are able to accept and distribute incoming NLB traffic to the rest of the cluster, creating potential ingress traffic bottlenecks and reduced internal load balancing capability, or even complete loss of ingress traffic incoming to the cluster if the subset of nodes with Ingress Gateway Pods goes down. See the source IP for Services with `Type=NodePort` for more information. For reference, see Istio [Network Load Balancer](https://istio.io/latest/docs/tasks/security/authorization/authz-ingress/#network) documentation.

Default Kyma Istio installation profile configures **PodAntiAffinity** to ensure that Ingress Gateway Pods are evenly spread across all Nodes. This guarantees that the above requirement is satisfied if your IngressGateway autoscaling configuration **minReplicas** is at least equal to the number of Nodes. You can configure autoscaling options in the Istio custom resource using **spec.config.components.ingressGateway.k8s.hpaSpec.minReplicas**.

> [!WARNING]
> Deploy an Ingress Gateway Pod to multiple nodes if you enable `externalTrafficPolicy: Local` in production Deployments.

> [!TIP]
> While using GCP or Azure, you can find your load balancer's IP address in the field **status.loadBalancer.ingress** of the `ingress-gateway` Service.

Add **gatewayExternalTrafficPolicy** to the Istio custom resource:

<!-- tabs:start -->
#### **kubectl**
Run the following command:

  ```bash
  kubectl patch istios/default -n kyma-system --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy": "Local"}}}'
  ```


#### **Kyma Dashboard**
1. Navigate to **Cluster Details** and select **Modify Modules**.
2. Choose the Istio module and select **Edit**.
3. In the `General` section, set the Gateway external traffic policy to `Local`.
4. Select **Save**.
<!-- tabs:end -->

### Create a Workload for Verification

1. [Create an HttpBin workload](https://kyma-project.io/#/api-gateway/user/tutorials/01-00-create-workload).
2. Export the following values as environment variables.
    ```bash
    export DOMAIN_TO_EXPOSE_WORKLOADS={GATEWAY_DOMAIN}
    export GATEWAY={GATEWAY_NAMESPACE}/GATEWAY_NAME}
   ```
3. Expose the HttpBin workload using a VirtualService.
    ```bash
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

### Verify the X-Forwarded-For and X-Envoy-External-Address Headers
1. Get your public IP address.
    ```bash
    curl -s https://api.ipify.org
    ```

2. Send a request to the HttpBin workload.
    ```bash
    curl -s "https://httpbin.$DOMAIN_TO_EXPOSE_WORKLOADS/get?show_env=true"
    ```
3. Verify that the response contains the **X-Forwarded-For** and **X-Envoy-External-Address** headers with your public IP address, for example:
    ```json
    {
      "args": {
        "show_env": "true"
      },
      "headers": {
        "Accept": "...",
        "Host": "...",
        "User-Agent": "...",
        "X-Envoy-Attempt-Count": "...",
        "X-Envoy-External-Address": "165.1.187.197",
        "X-Forwarded-Client-Cert": "...",
        "X-Forwarded-For": "165.1.187.197",
        "X-Forwarded-Proto": "...",
        "X-Request-Id": "..."
      },
      "origin": "165.1.187.197",
      "url": "..."
    }
    ``` 
