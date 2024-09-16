# Enable Automatic Istio Sidecar Proxy Injection

Enabling automatic sidecar injection allows `istiod` to watch all creation operations of the Pods on all namespaces, which are part of the Istio service mesh, and inject the newly created Pods with a sidecar proxy.

You can enable sidecar proxy injection for either an entire namespace or a single Deployment. Adding the `istio-injection=enabled` label on the namespace level results in injecting sidecars to all newly created Pods inside of the namespace. If the sidecar proxy injection is disabled at the namespace level, or the `sidecar.istio.io/inject` label on a Pod is set to `false`, the sidecar proxy is not injected.


## Enable Istio Sidecar Proxy Injection for a Namespace

  <!-- tabs:start -->
  #### **Kyma Dashboard**
  
  1. Select the namespace where you want to enable sidecar proxy injection.
  2. Choose **Edit**.
  3. In the `Form` section, siwtch the toggle to enable Istio sidecar proxy injection.
  4. Choose **Save**.

  #### **kubectl**
  
  Use the following command:
  
  ```bash
  kubectl label namespace {YOUR_NAMESPACE} istio-injection=enabled
  ```
  <!-- tabs:end -->


## Enable Istio Sidecar Proxy Injection for a Pod

  <!-- tabs:start -->

  #### **Kyma Dashboard**

  1. Select the namespace of the Pod's Deployment.
  2. In the **Workloads** section, select **Deployments**.
  3. Select the Pod's Deployment and click **Edit**.
  4. In the `UI Form` section, set the **sidecar.istio.io/inject** label to `true`.
  ![Switch the toggle to enable Istio sidecar injection](../../assets/sidecar-injection-toggle-deployment.svg)
  1. Click **Update**.

  #### **kubectl**
  
  Run the following command:
  
  ```bash
  kubectl patch -n {YOUR_NAMESPACE} deployments/{YOUR_DEPLOYMENT} -p '{"spec":{"template":{"metadata":{"labels":{"sidecar.istio.io/inject":"true"}}}}}'
  ```

  <!-- tabs:end -->