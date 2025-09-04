# Istio Proxy as Native Sidecar Container

## Differences Between Regular and Native Sidecar Containers

Every Pod in the Istio mesh gets an additional `istio-proxy` sidecar container, which intercepts network traffic to provide Istio features.

In the past, Kubernetes didn't support the concept of sidecar containers, so it wasn't possible to control the order in which containers within a Pod were started or stopped. This approach causes multiple issues, such as:
- When an application container starts before the `istio-proxy` sidecar container is ready, the application doesn't have network access at startup.
- When the `istio-proxy` sidecar stops before the application container stops, the application loses network access during shutdown.
- An init container runs before the `istio-proxy` sidecar container, so it can't access the network.
- A running `istio-proxy` sidecar container prevents Pods from being finished (for example, in the case of Jobs).

Since version 1.28, Kubernetes has offered [native sidecar containers](https://kubernetes.io/blog/2023/08/25/native-sidecar-containers/) functionality to address such problems. If an init container has **restartPolicy** set to `Always`, Kubernetes considers it a native sidecar and treats it differently.
Istio uses native sidecars introduced by Kubernetes for `istio-proxy` sidecars. See [Kubernetes Native Sidecars in Istio](https://istio.io/latest/blog/2023/native-sidecars/).

## Default Istio Proxy Sidecar Type

The Istio module configures the default type of the `istio-proxy` sidecar container, which is injected into all Pods that allow for Istio sidecar proxy injection. However, you can override this default setting for a specific Pod, allowing it to use another type of sidecar container regardless of the Istio module’s default configuration. The default `istio-proxy` sidecar type varies depending on the particular Istio module and Istio version:
- Istio module up to 1.20 - native sidecars are disabled by default
- Istio module 1.21 with Istio 1.27 - native sidecars are disabled by default
- Istio module 1.22 with Istio 1.27 - native sidecars are enabled by default, unless you set **compatibilityMode** in the Istio CR to `true`
- Istio module with Istio 1.28 - native sidecars are enabled by default

## Configuring the Type of Istio Sidecar Proxy for a Particular Workload

You can configure the sidecar type explicitly for a particular workload.
- To inject `istio-proxy` as a native sidecar container, set the `sidecar.istio.io/nativeSidecar` annotation to `"true"` on a given Pod or in the Pod template.
- To inject `istio-proxy` as a regular sidecar container, set the `sidecar.istio.io/nativeSidecar` annotation to `"false"` on a given Pod or in the Pod template.

If you do not set the annotation, the default setting is used.

You must set the annotation at the Pod level. So, when a Pod is created by a parent resource (for example, Deployment, StatefulSet, ReplicaSet, DaemonSet, Job, CronJob), you must configure the annotation in the Pod template. See the example:

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

Setting the sidecar type explicitly is especially recommended when a given workload works well only with one sidecar type and causes issues otherwise. This approach guarantees that the sidecar type doesn't change after the Istio module upgrade. A good example are Job Pods, which can't finish if the main container finishes because the `istio-proxy` sidecar container is still running. In this case, setting the `sidecar.istio.io/nativeSidecar` annotation in the Pod template solves the problem.

## Support in the Istio Module

The Istio module fully supports both types of sidecar containers. In particular, in case of Istio version update it detects which workloads should be restarted and this mechanism works with both sidecar types.

## Example of Istio Proxy as Native Sidecar

1. Create a `test` namespace with Istio injection enabled:

    ```bash
    kubectl create ns test
    kubectl label namespace test istio-injection=enabled
    ```

2. Create a workload with istio-proxy running as a native sidecar container:

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

3. To observe the progress, run:
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

The key differences when compared to the `istio-proxy` run as a regular sidecar container:
- When the Pod starts, it has the `Init` status with two containers.
- The application container only starts after the `istio-proxy` native sidecar is fully operational.
- After the main container completes its job, the Pod finishes with the status `Completed`.

4. List init containers:

    ```
    kubectl get pod -n test -o jsonpath='{.items[0].spec.initContainers[*].name}'
    ```
The output contains two containers, and `istio-proxy` is now an init container:
    
    ```
    istio-validation istio-proxy
    ```

5. List regular containers:

    ```
    kubectl get pod -n test -o jsonpath='{.items[0].spec.containers[*].name}'
    ```

    The output includes only the application container without `istio-proxy`:

    ```
    test-job
    ```

## Known Hacks that Native Sidecars Can Replace

### The /quitquitquit call

The solution was proposed under [issue #6324](https://github.com/istio/istio/issues/6324#issuecomment-533923427).

The `istio-proxy` sidecar offers an additional endpoint `/quitquitquit` that you can use in the following way to shut down `istio-proxy`:

```
curl -sf -XPOST http://127.0.0.1:15020/quitquitquit
```

Using the endpoint is no longer required if `istio-proxy` runs as a native sidecar. Its lifecycle is bound to the main application container, so the `istio-proxy` sidecar shuts down automatically when the main container is finished.

### runAsUser 1337, or excludeOutboundIPRanges, or excludeOutboundPorts

The solution was proposed in the blog post [Upcoming breaking change in SAP BTP, Kyma Runtime: Enabling the Istio CNI plugin](https://community.sap.com/t5/technology-blog-posts-by-sap/upcoming-breaking-change-in-sap-btp-kyma-runtime-enabling-the-istio-cni/ba-p/13550765).

Init containers are started before regular containers are started. This means that the `istio-proxy` as regular sidecar doesn't work when init containers are running. As a result init containers don't have network access.

By configuring **UID 1337**, **excludeOutboundIPRanges**, or **excludeOutboundPorts** you can exclude the network traffic from being captured by the `istio-proxy`. This allows init containers to access the network, but they are able to connect only to resources outside service mesh.

This is no longer required if `istio-proxy` runs as a native sidecar because the `istio-proxy` is injected as the first init container and runs until the main application container finishes. So, init containers are able to access the network in the same way as regular application containers, and they may connect to resources inside and outside the service mesh.

## Related Links

- [Kubernetes blog post about native sidecars](https://kubernetes.io/blog/2023/08/25/native-sidecar-containers/)
- [Istio blog post about native sidecars](https://istio.io/latest/blog/2023/native-sidecars/)
