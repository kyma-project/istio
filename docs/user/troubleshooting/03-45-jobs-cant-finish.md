<!-- open-source-only -->
# Pods Created by Jobs can't Finish

## Symptom

Pods created by Jobs remain stuck in 'NotReady' status after main containers finished.

## Cause

The istio-proxy run as regular sidecar container runs independently of the application container. There is no mechanism that shuts down the istio-proxy if the main container finished. Consequently, the pod is also running.

## Solution

Switch istio-proxy to be injected as native sidecar container.

Set `sidecar.istio.io/nativeSidecar` annotation in the pod template to `"true"`

Example:

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

This annotation instructs Istio to run istio-proxy as native sidecar container. Its lifecycle is then dependent on the main application container so it finishes automatically when the main application container is finished and the Pod state should be 'Completed' after that.

## More information

[Istio Proxy as Native Sidecar Container](../00--istio-proxy-as-native-sidecar.md)
