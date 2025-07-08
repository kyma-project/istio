# Exposing Workloads Using Istio VirtualService
Learn how to expose the workload with Istio [VirtualService](https://istio.io/latest/docs/reference/config/networking/virtual-service/) and your custom domain.

## Prerequisites
- You have the Istio module added.
- You have a deployed workload.
- To use CLI instructions, you must install kubectl and curl.
- You have set up your custom domain.

## Context
Kyma's API Gateway module provides the APIRule custom resource, which is the recommanded solution for securly exposing workloads. To use APIRules in Kyma, you must add the APIGateway module to your Kyma cluster. Additionally, APIRule in the newest version v2, requires the exposed workload to be in the Istio service mesh.

However, if you do not require the benefits provided by the Istio service mesh (such as secure service-to-service communication, tracing capabilities, or traffic management) and do not need additional authentication setup, you can expose an unsecured workload using Istio VirtualService only. Such approach might be useful in the following scenarios:
* If you use [Unified Gateway](https://pages.github.tools.sap/unified-gateway/) as an entry point for SAP Cloud solutions. In such a case the Unified Gateway's layer is responsible for the exposure of applications and APIs, and securing SAP endpoints.
* If you want to expose frontend of an application which manages authentication at the UI level.
* For development and testing purposes.


## Procedure

<!-- tabs:start -->
#### **Kyma Dashboard**
1. Go to the namespace in which you want to create a VirtualService resource.
2. Go to **Istio > Virtual Services**.
2. Choose **Create** and provide the following configuration details:
   
    Option | Description
    ---------|----------
    **Name** | The name of the VirtualService resource you're creating.
    **Hosts** | The fully qualified domain name (FQDN) of the service within the cluster. This is constructed using the subdomain and domain name.
    **Gateways**| The namespace which includes the Gateway you want to use. The name of the Gateway you want to use. Use the format ...

3. Go to **HTTP > Matches > Match** and provide the following configuration details:
   
    Option | Description
    ---------|----------
    **URI** | The URI prefix used to match incoming requests. This determines which requests will be routed to the specified destination.
    **Port** | This is the port number on which the destination service is listening. It specifies where the VirtualService will route the incoming traffic.
    
4. Go to **Routes > Route > Destinations > Host** and add the port number on which the destination service is listening. It specifies where the VirtualService will route the incoming traffic.

    
See the following example of a VirtualService resource that exposes an HTTPBin Service:

    ```yaml
    apiVersion: networking.istio.io/v1alpha3
    kind: VirtualService
    metadata:
        name: example-vs
        namespace: default
    spec:
        hosts:
        - httpbin.my-domain.com
        gateways:
        - kyma-system/my-gateway
        http:
        - match:
            - uri:
                prefix: /
            route:
            - destination:
                port:
                    number: 80
                    host: httpbin.my-domain.com.svc.cluster.local
    ```

#### **kubectl**

1. To expose a workload using VirtualService, replace the placeholders and run the following command:
   
    Option | Description
    ---------|----------
    **{VS_NAME}** | The name of the VirtualService resource you're creating.
    **{NAMESPACE}** | The namespace in which you want to create the VirtualService resource. 
    **{SUBDOMAIN}.{DOMAIN_NAME}** | The fully qualified domain name (FQDN) of the service within the cluster. This is constructed using the subdomain and domain name.
    **{GATEWAY_NAMESPACE}** | The namespace which includes the Gateway you want to use.
    **{GATEWAY_NAME}** | The name of the Gateway you want to use.
    **{PATH_PREFIX}** | The URI prefix used to match incoming requests. This determines which requests will be routed to the specified destination.
    **{PORT_NUMBER}** | This is the port number on which the destination service is listening. It specifies where the VirtualService will route the incoming traffic.

    For more configuration options, see [Virtual Service](https://istio.io/latest/docs/reference/config/networking/virtual-service/).

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: networking.istio.io/v1alpha3
    kind: VirtualService
    metadata:
        name: {VS_NAME}
        namespace: {NAMESPACE}
    spec:
        hosts:
            - {SUBDOMAIN}.{DOMAIN_NAME}
        gateways:
            - {GATEWAY_NAMESPACE}/{GATEWAY_NAME}
    http:
    - match:
        - uri:
            prefix: {PATH_PREFIX}
        route:
        - destination:
            port:
                number: {PORT_NUMBER}
                host: {SUBDOMAIN}.{DOMAIN_NAME}.svc.cluster.local
    EOF
    ```

    See the following example:

    ```yaml
    apiVersion: networking.istio.io/v1alpha3
    kind: VirtualService
    metadata:
    name: example-vs
    namespace: default
    spec:
    hosts:
    - httpbin.my-domain.com
    gateways:
    - kyma-system/my-gateway
    http:
    - match:
        - uri:
            prefix: /
        route:
        - destination:
            port:
            number: 80
            host: httpbin.my-domain.com.svc.cluster.local
    ```
2. To access the secured resources, run the following command:

    ```bash

    ```

<!-- tabs:end -->