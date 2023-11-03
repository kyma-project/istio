# Enable automatic Istio sidecar proxy injection

Enabling automatic sidecar injection allows `istiod` to watch all Pod creation operations on all Namespaces, which should be part of Istio Service Mesh, and inject the newly created Pods with a sidecar proxy.

You can enable sidecar proxy injection for either an entire Namespace or a single Deployment.

* Follow the steps to enable sidecar proxy injection for a Namespace:
  
  <!-- tabs:start -->

  #### **kubectl**
  
  Use the following command:
  
  ```bash
  kubectl label namespace {YOUR_NAMESPACE} istio-injection=enabled
  ```

  #### **Kyma Dashboard**
  
  1. Select the Namespace where you want to enable sidecar proxy injection.
  2. Click the blue **Edit** button.
  3. In the **UI Form** section, toggle the switch to set the **istio-injection** label value to `enabled` for the Namespace.
  4. Click **Update**.

     <!-- tabs:end --><br>

* Follow the steps to enable sidecar proxy injection for a Deployment:

  <!-- tabs:start -->

  #### **kubectl**
  
  Use the following command:
  
  ```bash
  kubectl label deployment {YOUR_DEPLOYMENT} istio-injection=enabled
  ```

  #### **Kyma Dashboard**

  1. Select the Namespace of the Deployment for which you want to enable sidecar proxy injection.
  2. Locate and select the Deployment.
  3. Click the blue **Edit** button.
  4. In the `UI Form` section, toggle the switch to set the **istio-injection** label value to `enabled` for the Deployment.
  5. Click **Update**.

  <!-- tabs:end -->

Note that if the sidecar proxy injection is disabled at the Namespace level or the `sidecar.istio.io/inject` label on a Pod is set to `false`, the sidecar proxy is not injected.

## Check whether your workloads have automatic Istio sidecar injection enabled

Check whether your workloads have automatic Istio sidecar injection enabled by running [the script](../../../assets/sidecar-analysis.sh). You can either pass the **namespace** parameter to the script or run it with no parameter.

* If you don't provide any parameter, the execution output contains Pods from all Namespaces that don't have automatic Istio sidecar injection enabled. The script outputs the information in the format of `{namespace}/{pod}`. Run:

    ```bash
    ./sidecar-analysis.sh
    ```
  
  You get an output similar to this one:

    ```
    Pods out of istio mesh:
      In namespace labeled with "istio-injection=disabled":
        - sidecar-disabled/some-pod
      In namespace labeled with "istio-injection=enabled" with pod labeled with "sidecar.istio.io/inject=false":
        - sidecar-enabled/some-pod
      In not labeled ns with pod not labeled with "sidecar.istio.io inject=true":
        - no-label/some-pod
    ```

*  If you pass a parameter, only the Pods from the specified Namespace are analyzed. The script outputs the information in the format of `{pod}` if run for a specific Namespace. Run:

    ```bash
    ./sidecar-analysis.sh {namespace}
    ```
  You get an output similar to this one:

  ```
  Pods out of istio mesh in namespace {namespace-name}:
    - some-pod
  ```