# Init Containers can't Access Network

## Symptom

Init containers can't access network.

## Cause

If istio injection is enabled the whole network traffic from all containers is intercepted by the `istio-proxy` container. Init containers are started before regular containers are started. This means that the `istio-proxy` as regular sidecar doesn't work when init containers are running. As a result init containers don't have network access.

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

This annotation instructs Istio to run `istio-proxy` as a native sidecar container. In this case the Istio injects the `istio-proxy` as the first init container, so all containers running later are able to access the network.

## Related Links

[Istio Proxy as Native Sidecar Container](../00--istio-proxy-as-native-sidecar.md)
