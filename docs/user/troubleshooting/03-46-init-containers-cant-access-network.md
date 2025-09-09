# Init Containers Can't Access the Network

## Symptom

Init containers can't access the network.

## Cause

If Istio injection is enabled, the `istio-proxy` container intercepts all network traffic from all containers. Init containers are started before regular containers. This means that `istio-proxy` running as a regular sidecar doesn't work when init containers are running. As a result, init containers don't have network access.

## Solution

Inject `istio-proxy` as a native sidecar container.

To do this, set the `sidecar.istio.io/nativeSidecar` annotation in the Pod to `"true"`.

See the following example:

```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: init-container-network-check
  namespace: test
  annotations:
    sidecar.istio.io/nativeSidecar: "true"
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

This annotation instructs Istio to run `istio-proxy` as a native sidecar container. In this case, Istio injects `istio-proxy` as the first init container, so all containers running later are able to access the network.

## Related Links

[Istio Proxy as Native Sidecar Container](../00-20-istio-proxy-as-native-sidecar.md)
