# Forward a Client IP in the X-Forwarded-For Header

Learn what the X-Forwarded-For (XFF) header is. Use it to forward client attributes to destination workloads.

## Prerequisites

* You have the Istio module added.
* To use CLI instruction, you must install [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) and [curl](https://curl.se/). Alternatively, you can use Kyma dashboard.
* You have Istio Ingress Gateway set up. See [Set Up a TLS Gateway in Simple Mode](https://kyma-project.io/#/api-gateway/user/tutorials/01-20-set-up-tls-gateway).

## Context

To function correctly, many applications must be provided with the client IP address of an originating request. This is often needed in cases where a workload must restrict access based on the client IP address. Reverse proxies pass client attributes to destination workloads using the XFF header.

The XFF header conveys the client IP address and the chain of intermediary proxies that the request traverses to reach the Istio service mesh. The header might not include all IP addresses if an intermediary proxy does not support modifying the header. Due to [technical limitations of AWS Classic ELBs](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-proxy-protocol.html#proxy-protocol), when using an IPv4 connection, the header does not include the public IP of the load balancer in front of Istio Ingress Gateway. Moreover, Istio Ingress Gateway's Envoy does not append the private IP address of the load balancer to the XFF header, effectively removing this information from the request. For more information on XFF, see the [IETF’s RFC documentation](https://datatracker.ietf.org/doc/html/rfc7239) and [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#x-forwarded-for).

To use the XFF header, you must configure the corresponding settings in the Istio custom resource (CR). 

## Procedure

<!-- tabs:start -->
#### **Kyma Dashboard**
1. Go to **Cluster Details**. 
2. Choose **Modify Modules**.
2. Select the Istio module.
3. Choose **Edit**.
4. In the `General` section, add the number of trusted proxies.
  Due to the variety of network topologies, the Istio CR must specify the number of trusted proxies deployed in front of the Istio Ingress Gateway proxy in the Istio CR, so that the client address can be extracted correctly.
6. If you use a GCP or Azure cluster, navigate to the Gateway section and set the Gateway traffic policy to Local. If you use a different cloud service provider, skip this step.
   >[!WARNING]
   > For production Deployments, it is strongly recommended to deploy Istio Ingress Gateway Pod to multiple nodes if you enable externalTrafficPolicy : Local. For more information, see Network Load Balancer.
   >
   >Default Istio installation profile configures PodAntiAffinity to ensure that Ingress Gateway Pods are evenly spread across all nodes. This guarantees that the above requirement is satisfied if your IngressGateway autoscaling configuration minReplicas is equal to or greater than the number of nodes. You can configure autoscaling options in the Istio CR using the field spec.config.components.ingressGateway.k8s.hpaSpec.minReplicas.<br>
   
   >[!TIP]
   > If you use a GCP or Azure cluster, you can find your load balancer's IP address in the field status.loadBalancer.ingress of the ingress-gateway Service.
7. Choose **Save**.


#### **kubectl**
1. In the following command, replace the placeholder with the number of trusted proxies deployed in front of the Istio Ingress Gateway proxy. Run the command:
    ```bash
    kubectl patch istios/default -n kyma-system --type merge -p '{"spec":{"config":{"numTrustedProxies": NUM_OF_TRUSTED_PROXIES}}}'
    ```
    Due to the variety of network topologies, the Istio CR must specify the configuration property **numTrustedProxies**, so that the client IP address can be extracted correctly.

2. If you use a GCP or Azure cluster, run the following command to set the traffic policy to Local. If you use a different cloud service provider, skip this step.

    ```bash
    kubectl patch istios/default -n kyma-system --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy": "Local"}}}'
    ```
    >[!WARNING]
    > For production Deployments, it is strongly recommended to deploy Istio Ingress Gateway Pod to multiple nodes if you `enable externalTrafficPolicy : Local`. For more information, see [Network Load Balancer](https://istio.io/latest/docs/tasks/security/authorization/authz-ingress/#network).
    >
    >Default Istio installation profile configures **PodAntiAffinity** to ensure that Ingress Gateway Pods are evenly spread across all nodes. This guarantees that the above requirement is satisfied if your IngressGateway autoscaling configuration **minReplicas** is equal to or greater than the number of nodes. You can configure autoscaling options in the Istio CR using the field **spec.config.components.ingressGateway.k8s.hpaSpec.minReplicas**.
    >[!TIP]
    > If you use a GCP or Azure cluster, you can find your load balancer's IP address in the field **status.loadBalancer.ingress of the ingress-gateway** Service.
<!-- tabs:end -->

### Results
When you call an exposed workload, the response contains the **X-Forwarded-For** and **X-Envoy-External-Address** headers with your public IP address. See an example response for the Client IP `165.1.187.197`:
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

>[!TIP]
> You can check your public IP address at https://api.ipify.org.
