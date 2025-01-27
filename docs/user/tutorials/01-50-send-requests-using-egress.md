# Send Requests Using Istio Egress Gateway
Learn how to configure and use the Istio egress Gateway to allow outbound traffic from your Kyma runtime cluster to specific external destinations. Test your configuration by sending an HTTPS request to an external website using a sample Deployment.

## Prerequisites

* You have the Istio module added.
* To use CLI instruction, you must install [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
  and [curl](https://curl.se/).

## Steps

1. Export the following value as an environment variable:

    ```bash
    export NAMESPACE={service-namespace}
    ```

2. Create a new namespace for the sample application:
    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    ```

3. Enable the egress Gateway in the Istio custom resource:
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: operator.kyma-project.io/v1alpha2
   kind: Istio
   metadata:
     name: default
     namespace: kyma-system
     labels:
       app.kubernetes.io/name: default
   spec:
     components:
       egressGateway:
         enabled: true
   EOF
   ```

4. Enable additional sidecar logs to see the egress Gateway being used in requests:
    ```bash
    kubectl apply -f - <<EOF
    apiVersion: telemetry.istio.io/v1
    kind: Telemetry
    metadata:
      name: mesh-default
      namespace: istio-system
    spec:
      accessLogging:
        - providers:
          - name: envoy
    EOF
    ```

5. Apply the `curl` Deployment to send the requests:
    ```bash
    kubectl apply -f - <<EOF
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: curl
      namespace: ${NAMESPACE}
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: curl
      namespace: ${NAMESPACE}
      labels:
        app: curl
        service: curl
    spec:
      ports:
      - port: 80
        name: http
      selector:
        app: curl
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: curl
      namespace: ${NAMESPACE}
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: curl
      template:
        metadata:
          labels:
            app: curl
        spec:
          terminationGracePeriodSeconds: 0
          serviceAccountName: curl
          containers:
          - name: curl
            image: curlimages/curl
            command: ["/bin/sleep", "infinity"]
            imagePullPolicy: IfNotPresent
            volumeMounts:
            - mountPath: /etc/curl/tls
              name: secret-volume
          volumes:
          - name: secret-volume
            secret:
              secretName: curl-secret
              optional: true
    EOF
    ```

   Export the name of the `curl` Pod:
    ```bash
   export SOURCE_POD=$(kubectl get pod -n "$NAMESPACE" -l app=curl -o jsonpath={.items..metadata.name})
    ```

6. Define a ServiceEntry to allow outbound traffic to the `kyma-project` domain and perform DNS resolution:
   
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: networking.istio.io/v1
   kind: ServiceEntry
   metadata:
     name: kyma-project
     namespace: $NAMESPACE
   spec:
     hosts:
     - kyma-project.io
     ports:
     - number: 443
       name: tls
       protocol: TLS
     resolution: DNS
   EOF
   ```

7. Create an egress Gateway, DestinationRule, and VirtualService to direct traffic:
   
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: networking.istio.io/v1
   kind: Gateway
   metadata:
     name: istio-egressgateway
     namespace: ${NAMESPACE}
   spec:
     selector:
       istio: egressgateway
     servers:
     - port:
         number: 443
         name: tls
         protocol: TLS
       hosts:
       - kyma-project.io
       tls:
         mode: PASSTHROUGH
   ---
   apiVersion: networking.istio.io/v1
   kind: DestinationRule
   metadata:
     name: egressgateway-for-kyma-project
     namespace: ${NAMESPACE}
   spec:
     host: istio-egressgateway.istio-system.svc.cluster.local
     subsets:
     - name: kyma-project
   ---
   apiVersion: networking.istio.io/v1
   kind: VirtualService
   metadata:
     name: direct-kyma-project-through-egress-gateway
     namespace: ${NAMESPACE}
   spec:
     hosts:
     - kyma-project.io
     gateways:
     - mesh
     - istio-egressgateway
     tls:
     - match:
       - gateways:
         - mesh
         port: 443
         sniHosts:
         - kyma-project.io
       route:
       - destination:
           host: istio-egressgateway.istio-system.svc.cluster.local
           subset: kyma-project
           port:
             number: 443
     - match:
       - gateways:
         - istio-egressgateway
         port: 443
         sniHosts:
         - kyma-project.io
       route:
       - destination:
           host: kyma-project.io
           port:
             number: 443
         weight: 100
   EOF
   ```
   
8. Send an HTTPS request to the Kyma project website:
   ```bash
   kubectl exec -n "$NAMESPACE" "$SOURCE_POD" -c curl -- curl -sSL -o /dev/null -D - https://kyma-project.io
   ```
   
   If successful, you get a response from the website similar to this one:
   ```
   HTTP/2 200
   accept-ranges: bytes
   age: 203
   ...
   ```
   
   Check the logs of the Istio egress Gateway:
   ```bash
   kubectl logs -l istio=egressgateway -n istio-system
   ```

   You should see the request made by the egress Gateway in the logs:
   ```
   {"requested_server_name":"kyma-project.io","upstream_cluster":"outbound|443||kyma-project.io",[...]}
   ```
