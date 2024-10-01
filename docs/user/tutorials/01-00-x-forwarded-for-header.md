# Forward a Client IP in the X-Forwarded-For Header

Many applications must be provided with the client IP address of an originating request to funtion correctly. This is often needed in cases where a workload must restrict access based on the client's IP address. The ability to provide client attributes to services has long been a staple of reverse proxies, which use the X-Forwarded-For (XFF) header to forward client attributes to destination workloads.

The XFF header conveys the client IP address and the chain of intermediary proxies that the request traverse to reach the Istio service mesh.
The header might not include all IP addresses if an intermediary proxy does not support modifying the header.
Due to [technical limitations of AWS Classic ELBs](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-proxy-protocol.html#proxy-protocol), when using an IPv4 connection, the header does not include the public IP of the load balancer in front of Istio Ingress Gateway.
Moreover, Istio Ingress Gateway Envoy does not append the private IP address of the load balancer to the XFF header, effectively removing this information from the request. For more information on XFF, see 
the [IETF’s RFC documentation](https://datatracker.ietf.org/doc/html/rfc7239) and [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#x-forwarded-for). 

## Prerequisites

* The Istio module added or [Kyma Istio Operator](../../../README.md#install-kyma-istio-operator-and-istio-from-the-latest-release) installed
* [Istio Ingress Gateway](https://kyma-project.io/#/api-gateway/user/tutorials/01-20-set-up-tls-gateway) set up

## Steps

### Configure the Number of Trusted Proxies in the Istio Custom Resource

Applications rely on reverse proxies to forward the client IP address in a request using the XFF header. 
Due to the variety of network topologies, you must specify the configuration property **numTrustedProxies** in the Istio custom resource, so that the client address can be extracted correctly. This property indicates the number of trusted proxies deployed in front of the Istio Ingress Gateway proxy.

Add **numTrustedProxies** to the Istio custom resource:

<!-- tabs:start -->
#### **kubectl**
Run the following command:

  ```bash
  kubectl patch istios/default -n kyma-system --type merge -p '{"spec":{"config":{"numTrustedProxies": NUM_OF_TRUSTED_PROXIES}}}'
  ```

#### **Kyma Dashboard**
1. Navigate to **Cluster Details**. 
2. Select **Modify Modules**.
2. Choose the Istio module.
3. Select **Edit**.
4. In the `General` section, add the number of trusted proxies.
5. Select **Save**.
<!-- tabs:end -->

### Configure Gateway External Traffic Policy in the Istio Custom Resource (GCP and Azure only)

If you are using a GCP or Azure cluster, you must also set the **gatewayExternalTrafficPolicy** to `Local` in order to include the client's IP address in the XFF header. Skip this step if you're using a different cloud service provider.

For production Deployments, it is strongly recommended to deploy an Ingress Gateway Pod to multiple nodes if you enable `externalTrafficPolicy: Local`. For more information, see [Network Load Balancer](https://istio.io/latest/docs/tasks/security/authorization/authz-ingress/#network).

Default Istio installation profile configures **PodAntiAffinity** to ensure that Ingress Gateway Pods are evenly spread across all Nodes. This guarantees that the above requirement is satisfied if your IngressGateway autoscaling configuration **minReplicas** is at least equal to the number of Nodes. You can configure autoscaling options in the Istio custom resource using **spec.config.components.ingressGateway.k8s.hpaSpec.minReplicas**.

> [!WARNING]
> Deploy an Ingress Gateway Pod to multiple nodes if you enable `externalTrafficPolicy: Local` in production Deployments.
> Enabling `externalTrafficPolicy: Local` may result in a temporal increase in request delay. Make sure that this is acceptable.

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

1. In the `Cluster Details` section, select `+ Upload YAML`.
2. Paste the following configuration into the editor:
    ```yaml
    apiVersion: v1
    kind: Namespace
    metadata:
      name: httpbin
      labels:
        istio-injection: enabled
    ---   
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: httpbin
      namespace: httpbin
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: httpbin
      namespace: httpbin
      labels:
        app: httpbin
        service: httpbin
    spec:
      ports:
      - name: http
        port: 8000
        targetPort: 80
      selector:
        app: httpbin
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: httpbin
      namespace: httpbin
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: httpbin
          version: v1
      template:
        metadata:
          labels:
            app: httpbin
            version: v1
        spec:
          serviceAccountName: httpbin
          containers:
          - image: docker.io/kennethreitz/httpbin
            imagePullPolicy: IfNotPresent
            name: httpbin
            ports:
            - containerPort: 80
    ---
    apiVersion: networking.istio.io/v1alpha3
    kind: VirtualService
    metadata:
      name: httpbin
      namespace: httpbin
    spec:
      hosts:
      - "httpbin.{DOMAIN_TO_EXPOSE_WORKLOADS}"
      gateways:
      - {GATEWAY}
      http:
      - match:
        - uri:
            prefix: /
        route:
        - destination:
            port:
              number: 8000
            host: httpbin.httpbin.svc.cluster.local
    ```
    This code creates the namespace httpbin, deploys the HTTPBin Service in it, and exposes the HTTP Service using a VirtualService resource.
3. In the pasted configuration of the VirtualService resource replace the placeholders.
4. Select `Upload`.
5. Select `Close`.
6. Check your public IP adress at https://api.ipify.org.
7. Replace the placeholder with the name of your domain and follow the link: https://httpbin.{DOMAIN_TO_EXPOSE_WORKLOADS}/get?show_env=true.
8. Verify that the response contains the **X-Forwarded-For** and **X-Envoy-External-Address** headers with your public IP address.

-------------

1. To create a sample workload, run:
    
    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: v1
    kind: Namespace
    metadata:
      name: httpbin
      labels:
        istio-injection: enabled 
    ---   
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: httpbin
      namespace: httpbin
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: httpbin
      namespace: httpbin
      labels:
        app: httpbin
        service: httpbin
    spec:
      ports:
      - name: http
        port: 8000
        targetPort: 80
      selector:
        app: httpbin
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: httpbin
      namespace: httpbin
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: httpbin
          version: v1
      template:
        metadata:
          labels:
            app: httpbin
            version: v1
        spec:
          serviceAccountName: httpbin
          containers:
          - image: docker.io/kennethreitz/httpbin
            imagePullPolicy: IfNotPresent
            name: httpbin
            ports:
            - containerPort: 80
    EOF
    ```

1. Export the following values as environment variables.
    ```bash
    export DOMAIN_TO_EXPOSE_WORKLOADS={GATEWAY_DOMAIN}
    export GATEWAY={GATEWAY_NAMESPACE/GATEWAY_NAME}
   ```
2. Expose the HttpBin workload using a VirtualService.
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


Go to Cluster Details.
Select Modify Modules.
Choose the Istio module.
Select Edit.
In the General section, add the number of trusted proxies.
GCP and Azure: In the Gateway section, set the Gateway traffic policy to Local.
Select Save.
