# Pods Created by Jobs Can't Finish

## Symptom

Pods created by Jobs remain stuck in the `NotReady` status after the main containers have finished.

## Cause
By default, the Istio module injects `istio-proxy` containers as native sidecars. However, you can also set the annotation `sidecar.istio.io/nativeSidecar` to `"false"` for a specific Pod. This annotation overwrites the default setting and indicates that instead of native sidecars, the annotated Pod must be injected with a regular sidecar container.

When the `istio-proxy` container is a regular sidecar container, it runs independently of the application container. There is no mechanism that shuts down the `istio-proxy` sidecar when the main container completes its tasks. Consequently, the Pod is also running.

## Solution
Check if your Pod or the Pod's template contains the annotation `sidecar.istio.io/nativeSidecar: "false"`. See the following example of an annotated Job:

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
        sidecar.istio.io/nativeSidecar: "false"
    spec:
      containers:
      - name: test-job
        image: alpine:latest
        command: [ "/bin/sleep", "10" ]
      restartPolicy: Never
```

If the annotation is present, determine the reason for applying the annotation to your workload and, if possible, remove the annotation.

Removing the annotation `sidecar.istio.io/nativeSidecar: "false"` from a Pod template, allows the Istio module to run this container as a native sidecar. Then, the lifecycle of the native sidecar depends on the main application container, so `istio-proxy` finishes automatically at the same time as the main application container. After this, the Pod's state changes to `Completed`.


## Related Links

[Istio Proxy as Native Sidecar Container](../00-20-istio-proxy-as-native-sidecar.md)
