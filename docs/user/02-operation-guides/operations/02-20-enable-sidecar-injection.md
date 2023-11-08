# Enable automatic Istio sidecar proxy injection

Enabling automatic sidecar injection allows `istiod` to watch all Pod creation operations on all Namespaces, which should be part of Istio Service Mesh, and inject the newly created Pods with a sidecar proxy.

You can enable sidecar proxy injection for either an entire Namespace or a single Deployment.

>**WARNING:** Adding the `istio-injection=enabled` label on the Namespace level results in injecting sidecars to all Pods inside of the Namespace.

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
  
  <!-- tabs:end -->

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