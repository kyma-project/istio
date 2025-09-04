# Pods Created by Jobs can't Finish

## Symptom

Pods created by Jobs remain stuck in the `NotReady` status after the main containers have finished.

## Cause

The `istio-proxy` sidecar runs as a regular sidecar container, independently of the application container. There is no mechanism that shuts down the `istio-proxy` sidecar when the main container completes its tasks. Consequently, the Pod is also running.

## Solution

Inject `istio-proxy` as a native sidecar container.

To do this, set the `sidecar.istio.io/nativeSidecar` annotation in the Pod template to `"true"`.

See the following example:

```
apiVersion: batch/v1
kind: Job
metadata:
  name: test-job
  namespace: test
spec:
  template:
    metadata:
      annotations:
        sidecar.istio.io/nativeSidecar: "true"
    spec:
      containers:
      - name: test-job
        image: alpine:latest
        command: [ "/bin/sleep", "10" ]
      restartPolicy: Never
```

This annotation instructs Istio to run `istio-proxy` as a native sidecar container. Then, the lifecycle of the native sidecar depends on the main application container, so `istio-proxy` finishes automatically at the same time as the main application container. After this, the Pod's state changes to `Completed`.

## Related Links

[Istio Proxy as Native Sidecar Container](../00--istio-proxy-as-native-sidecar.md)
