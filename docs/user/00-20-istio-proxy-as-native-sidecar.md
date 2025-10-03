# Regular and Native Sidecar Containers
Understand the differences between regular and native sidecar containers, learn about the default settings applied by the Istio module, and find out how to override these settings for specific workloads as needed.

## Advantages of Native Sidecar Containers

Every Pod in the Istio mesh gets an additional `istio-proxy` sidecar container, which intercepts network traffic to provide Istio features. For sidecar proxies, Istio uses two types of sidecar containers: regular sidecars and native sidecars. Native sidecars are init containers that have **restartPolicy** set to `Always`. This approach, introduced with Kubernetes 1.28, allows for controlling the startup and shutdown sequence of containers within a Pod, which is not possible with sidecars deployed as regular containers.

Using native sidecars resolves the following problems that regular sidecars cause:
- When an application container starts before the `istio-proxy` sidecar container is ready, the application doesn't have network access at startup.
- When the `istio-proxy` sidecar stops before the application container stops, the application loses network access during shutdown.
- An init container runs before the `istio-proxy` sidecar container, so it can't access the network.
- A running `istio-proxy` sidecar container prevents Pods from being finished (for example, in the case of Jobs).

## Native Sidecars as the Default Istio Proxy Sidecar Type

By default, the Istio module 1.22 uses `istio-proxy` native sidecar containers. This setting is configured globally and applied to all Pods that allow for Istio sidecar proxy injection. However, you can override this default setting for a specific Pod, allowing it to use regular sidecar container instead.

Furthermore, if the Istio module deploys Istio version 1.27, you have the option to set the **compatibilityMode** in the Istio Custom Resource (CR) to `true`. This setting will make Istio behave as though it is version 1.26, which uses regular sidecar containers by default.

Istio module version | Istio version | Default Sidecar Type
---------|----------|---------
 1.20 or lower | 1.26 or lower | Regular sidecars
 1.21 | 1.27 | Regular sidecars
 1.22 | 1.27 | Native sidecars, unless you set **compatibilityMode** in the Istio CR to `true`
 One of the next versions after 1.22 with updated Istio | 1.28 | Native sidecars

## Using Regular Sidecar Containers for a Particular Workload

You can configure the sidecar type explicitly for a particular workload. To inject `istio-proxy` as a regular sidecar container, set the `sidecar.istio.io/nativeSidecar` annotation to `"false"` on a given Pod or in the Pod template. If you do not set the annotation, native sidecars are used.

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
        sidecar.istio.io/nativeSidecar: "false"
...
```

Setting the sidecar type explicitly is only recommended when a specific workload requires regular sidecar containers to function properly and experiences issues otherwise. This might be necessary if you have implemented workarounds to address problems caused by regular sidecars and now need additional time to adapt to native sidecars becoming the new default.

## Support in the Istio Module

The Istio module fully supports both types of sidecar containers. In particular, in case of Istio version update it detects which workloads should be restarted and this mechanism works with both sidecar types.

## Example: Differences Between Regular and Native Sidecars

1. Create a `test` namespace with Istio injection enabled:

    ```bash
    kubectl create ns test
    kubectl label namespace test istio-injection=enabled
    ```

2. Create a workload with `istio-proxy` running as a native sidecar container. 

    The Pod template does not contain the annotation `sidecar.istio.io/nativeSidecar: "false"`, so the `istio-proxy` is injected as a native sidecar container.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: test-job
      namespace: test
    spec:
      template:
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

The `istio-proxy` sidecar offers an additional endpoint `/quitquitquit` that you can use in the following way to shut down `istio-proxy`:

```
curl -sf -XPOST http://127.0.0.1:15020/quitquitquit
```
This solution was proposed under [issue #6324](https://github.com/istio/istio/issues/6324#issuecomment-533923427).

Using the endpoint is no longer required if `istio-proxy` runs as a native sidecar. Its lifecycle is bound to the main application container, so the `istio-proxy` sidecar shuts down automatically when the main container is finished.

### runAsUser 1337, or excludeOutboundIPRanges, or excludeOutboundPorts

Init containers are started before regular containers are started. This means that the `istio-proxy` as regular sidecar doesn't work when init containers are running. As a result init containers don't have network access.

By configuring **UID 1337**, **excludeOutboundIPRanges**, or **excludeOutboundPorts** you can exclude the network traffic from being captured by the `istio-proxy`. This allows init containers to access the network, but they are able to connect only to resources outside service mesh. The solution was proposed in the blog post [Upcoming breaking change in SAP BTP, Kyma Runtime: Enabling the Istio CNI plugin](https://community.sap.com/t5/technology-blog-posts-by-sap/upcoming-breaking-change-in-sap-btp-kyma-runtime-enabling-the-istio-cni/ba-p/13550765).

This is no longer required if `istio-proxy` runs as a native sidecar because the `istio-proxy` is injected as the first init container and runs until the main application container finishes. So, init containers are able to access the network in the same way as regular application containers, and they may connect to resources inside and outside the service mesh.

## Related Links

- [Kubernetes blog post about native sidecars](https://kubernetes.io/blog/2023/08/25/native-sidecar-containers/)
- [Istio blog post about native sidecars](https://istio.io/latest/blog/2023/native-sidecars/)
