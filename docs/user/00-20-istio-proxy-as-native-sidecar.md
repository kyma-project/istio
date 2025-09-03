# Istio Proxy as Native Sidecar Container

# Regular vs Native Sidecars Containers

Every pod in the Istio mesh is given an additional istio-proxy sidecar container, which intercepts the network traffic in order to provide Istio features.

In past the Kubernetes had no concept of 'sidecar' containers, so there was no way to control the order of starting or stopping containers within a pod. This may cause multiple issues, like:
- application container can start before istio-proxy sidecar is ready, so the application doesn't have the network access at startup
- istio-proxy may stop, before the application container is stopped, so the application loses the network access during shutdown
- init container runs before istio-proxy sidecar, so it can't access the network
- running istio-proxy container prevents pods from being finished (like in Jobs)

Kubernetes since 1.28 offers [native sidecar containers](https://kubernetes.io/blog/2023/08/25/native-sidecar-containers/) functionality to address such problems. An init container with restartPolicy set to Always is considered as native sidecar and Kubernetes treats it differently.
Istio [can use this feature](https://istio.io/latest/blog/2023/native-sidecars/) for istio-proxy sidecars.

## How to Configure Istio Proxy Sidecar Type for a Particular Workload

You can always configure the sidecar type explicitly for a particular workload.
- in order to inject istio-proxy as native sidecar container you need to set `sidecar.istio.io/nativeSidecar` annotation on a given pod or pod template to `true`
- in order to inject istio-proxy as regular sidecar container you need to set `sidecar.istio.io/nativeSidecar` annotation on a given pod or pod template to `false`

If the annotation is not set then the default setting is used.

The annotation must be set on a Pod level, so when a Pod is created by a parent resource (like Deployment, StatefulSet, ReplicaSet, DaemonSet, Job, CronJob, etc.) you need to configure the annotation in the pod template. See the example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  template:
    metadata:
      annotations:
        sidecar.istio.io/nativeSidecar: "true"
...
```

Setting the sidecar type explicitly is recommended especially when the given workload works well only with one sidecar type and causes issues otherwise. This approach guarantees that the sidecar type doesn't change after the Istio module upgrade. The good example are Job pods, which can't finish if the main container finishes, because the istio-proxy container still runs. In this case setting the `sidecar.istio.io/nativeSidecar` annotation on the pod template solves the problem.

## Default Istio Proxy Sidecar Type

The default istio-proxy sidecar type varies depending on the particular Istio module and Istio version:
- Istio module up to 1.20 - native sidecars are disabled by default
- Istio module 1.21 with Istio 1.27 - native sidecars are still disabled by default
- Istio module 1.22 with Istio 1.27 - native sidecars are enabled by default, unless compatibilityMode in the Istio CR is set to true
- Istio module 1.23 or newer with Istio 1.28 - native sidecars are always enabled by default

## Support in Istio Module

Istio module fully supports both types of sidecar containers. In particular, it is able to restart workloads with outdated istio-proxy version regardless if it runs as native sidecar or regular sidecar container.

## Example of Istio Proxy as Native Sidecar

Let's suppose that there is a 'test' namespace with istio injection enabled:

```bash
kubectl create ns test
kubectl label namespace test istio-injection=enabled
```

Let's now create a workload with istio-proxy running as native sidecar container:

```bash
kubectl apply -f - <<EOF
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
EOF
```

You can observe the progress:
```
kubectl get pod -n test -w
```

```
NAME             READY   STATUS     RESTARTS   AGE
test-job-2tt95   0/2     Init:0/2   0          0s
test-job-2tt95   0/2     Init:1/2   0          1s
test-job-2tt95   0/2     Init:1/2   0          2s
test-job-2tt95   0/2     PodInitializing   0          10s
test-job-2tt95   1/2     PodInitializing   0          10s
test-job-2tt95   2/2     Running           0          11s
test-job-2tt95   1/2     Completed         0          21s
test-job-2tt95   0/2     Completed         0          22s
```

The key differences when compared to the istio-proxy run as regular sidecar container:
- Pod starts with Init status with two containers
- application container is started after istio-proxy native sidecar is fully operational
- Pod finishes with status Completed after the main container finished its job

Let's also list init containers:

```
kubectl get pod -n test -o jsonpath='{.items[0].spec.initContainers[*].name}'
```
It shows two containers - istio-proxy is now an init container:

```
istio-validation istio-proxy
```

Let's now list regular containers:

```
kubectl get pod -n test -o jsonpath='{.items[0].spec.containers[*].name}'
```

It shows only the application container without istio-proxy:

```
test-job
```

## Known Hacks That Can Be Eliminated by Native Sidecars

### /quitquitquit call

It was proposed here: https://github.com/istio/istio/issues/6324#issuecomment-533923427

The istio-proxy offers an additional endpoint `/quitquitquit` that can be used to tell the istio-proxy to shut down in the following way:

```
curl -sf -XPOST http://127.0.0.1:15020/quitquitquit
```

This is not required anymore if the istio-proxy runs as native sidecar, because its lifecycle is bound with the main application container, so it shuts down automatically when the main container is finished.

### runAsUser 1337 or excludeOutboundIPRanges or excludeOutboundPorts

It was proposed here: https://community.sap.com/t5/technology-blog-posts-by-sap/upcoming-breaking-change-in-sap-btp-kyma-runtime-enabling-the-istio-cni/ba-p/13550765

Configuring UID 1337, excludeOutboundIPRanges or excludeOutboundPorts prevents the traffic from being redirected to the istio-proxy which hasn't been started yet.

This is not required anymore if the istio-proxy runs as native sidecar, because native sidecars are started before other init containers.

## More information

- [Kubernetes blog post about native sidecars](https://kubernetes.io/blog/2023/08/25/native-sidecar-containers/)
- [Istio blog post about native sidecars](https://istio.io/latest/blog/2023/native-sidecars/)
