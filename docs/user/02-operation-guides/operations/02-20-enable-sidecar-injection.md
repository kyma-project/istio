# Enable automatic Istio sidecar proxy injection

Enabling automatic sidecar injection allows `istiod` to watch all Pod creation operations on all Namespaces, which should be part of Istio Service Mesh, and inject the newly created Pods with a sidecar proxy.

You can enable sidecar proxy injection for either an entire Namespace or a single Deployment. If the sidecar proxy injection is disabled at the Namespace level or the `sidecar.istio.io/inject` label on a Pod is set to `false`, the sidecar proxy is not injected.

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
  2. Click the **Edit** button.
  3. In the `UI Form` section, toggle the switch to set the **istio-injection** label value to `enabled` for the Namespace.
  ![Switch the toggle to enable Istio sidecar injection](./../../../assets/sidecar-injection-toggle-namespace.svg)
  4. Click **Update**.
  <!-- tabs:end -->


* Follow the steps to enable sidecar proxy injection for a Pod:

  <!-- tabs:start -->

  #### **kubectl**
  
  Use the following command:
  
  ```bash
  kubectl patch -n {YOUR_NAMESPACE} deployments/{YOUR_DEPLOYMENT} -p '{"spec":{"template":{"metadata":{"labels":{"sidecar.istio.io/inject":"true"}}}}}'
  ```

  #### **Kyma Dashboard**

  1. Select the Namespace of the Pod's Deployment.
  2. On the left-hand side sidebar, find the **Workloads** section and select **Deployments**.
  3. Select the Pod's Deployment and click the **Edit** button.
  4. In the `UI Form` section, toggle the switch to set the **istio-injection** label value to `enabled` for the Deployment's Pods.
  ![Switch the toggle to enable Istio sidecar injection](./../../../assets/sidecar-injection-toggle-deployment.svg)
  1. Click **Update**.

  <!-- tabs:end -->