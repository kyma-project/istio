# Send requests using Istio Egress

## Prerequisites

* You have the Istio module added.
* To use CLI instruction, you must install [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
  and [curl](https://curl.se/). Alternatively, you can use Kyma dashboard.

### Configuration

1. Export the following values as environment variables:

    ```bash
    export NAMESPACE={service-namespace}
    ```

2. Create a new namespace for the sample application.
    ```bash
    kubectl create ns $NAMESPACE
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    ```

3. Enable additional sidecar logs to see egresGateway being used in requests:
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

4. Apply `curl` deployment to send the requests:
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

   Get the `curl` pod:
    ```bash
   export SOURCE_POD=$(kubectl get pod -n "$NAMESPACE" -l app=curl -o jsonpath={.items..metadata.name})
    ```

5. Define a `ServiceEntry`:
   
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
   
   Verify that the `ServiceEntry` was created successfully:
   ```bash
   kubectl exec -n "$NAMESPACE" "$SOURCE_POD" -c curl -- curl -sSL -o /dev/null -D - https://kyma-project.io
   ```

6. Create an egress `Gateway`, `DestinationRule` and `VirtualService` to direct traffic:

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

Once again, send an HTTPS reqest:
```bash
kubectl exec -n "$NAMESPACE" "$SOURCE_POD" -c curl -- curl -sSL -o /dev/null -D - https://kyma-project.io
```

Check egress gateway log:
```bash
kubectl logs -l istio=egressgateway -n istio-system
```
