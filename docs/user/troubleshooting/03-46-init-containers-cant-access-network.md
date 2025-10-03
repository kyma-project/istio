# Init Containers Can't Access the Network

## Symptom

Init containers can't access the network.

## Cause
If Istio injection is enabled, the `istio-proxy` container intercepts all network traffic from all containers. By default, the Istio module injects `istio-proxy` containers as native sidecars. However, you can also set the annotation `sidecar.istio.io/nativeSidecar` annotation to `"false"` for a specific Pod. This annotation overwrites the default setting and indicates that instead of native sidecars, the annotated Pod must be injected with a regular sidecar container.

Init containers are started before regular containers. If `istio-proxy` is a regular sidecar, it doesn't work when init containers are running. As a result, init containers don't have network access.

## Solution

Check if your Pod or the Pod's template contains the annotation `sidecar.istio.io/nativeSidecar: "false"`. See the following example of an annotated Pod:

```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: init-container-network-check
  namespace: test
  annotations:
    sidecar.istio.io/nativeSidecar: "false"
spec:
  initContainers:
  - name: init
    image: curlimages/curl
    command: [ "curl", "httpbin.org/get" ]
  containers:
  - name: main
    image: alpine:latest
    command: ["/bin/sleep", "10"]
  restartPolicy: Never
EOF
```

If the annotation is present, determine the reason for applying the annotation to your workload and, if possible, remove the annotation.

Removing the annotation `sidecar.istio.io/nativeSidecar: "false"` from a Pod template, allows the Istio module to run this container as a native sidecar. In this case, Istio injects `istio-proxy` as the first init container, so all containers running later are able to access the network.

## Related Links

[Istio Proxy as Native Sidecar Container](../00-20-istio-proxy-as-native-sidecar.md)
