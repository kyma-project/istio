# Can't access a Kyma endpoint (503 status code)

## Symptom

You try to access a Kyma endpoint and receive the `503` status code.

## Cause

This behavior might be caused by a configuration error in Istio Ingress Gateway. As a result, the endpoint you call is not exposed.

## Remedy

To fix this problem, restart the Pods of Istio Ingress Gateway.

<!-- tabs:start -->

#### **kubectl**

1. List all available endpoints:

    ```bash
    kubectl get virtualservice --all-namespaces
    ```

2. Delete the Pods of Istio Ingress Gateway to trigger the recreation of their configuration:

     ```bash
     kubectl delete pod -l app=istio-ingressgateway -n istio-system
     ```

#### **Kyma Dashboard**

1. Go to the `istio-system` namespace.
2. Navigate to the **Workloads** section on the left-hand side and click on **Pods**.
3. Use the search function to filter for all Pods labeled with `app=istio-ingressgateway`.
4. Delete each of the Pods that are displayed in order to trigger the recreation of their configuration.

<!-- tabs:end -->

If the restart doesn't help, change the image of Istio Ingress Gateway to allow further investigation. Kyma Istio Operator uses distroless Istio images that are more secure, but you cannot execute commands inside them. Follow these steps:

1. Edit the Istio Ingress Gateway Deployment:

   <!-- tabs:start -->

   #### **kubectl**
   1. Run the following command:
      ```bash
      kubectl edit deployment -n istio-system istio-ingressgateway
      ```
   2. Find the `istio-proxy` container and delete the `-distroless` suffix.
   
   #### **Kyma Dashboard**

   1. Go to the `istio-system` namespace.
   2. Navigate to the **Workloads** section on the left-hand side and select **Deployments**.
   3. Choose `istio-ingressgateway` and click the blue **Edit** button.
   4. Under the Containers toggle find the `istio-proxy` container. Delete the `-distroless` suffix from its **Docker image** field.
   5. Click **Update**.

   <!-- tabs:end -->


2. Check all ports used by Istio Ingress Gateway:

   <!-- tabs:start -->

   #### **kubectl**
   Run the following command:

   ```bash
   kubectl exec -ti -n istio-system $(kubectl get pod -l app=istio-ingressgateway -n istio-system -o name) -c istio-proxy -- netstat -lptnu
    ```

   #### **Kyma Dashboard**
   1. Go to the `istio-system` namespace.
   2. Navigate to the **Workloads** section on the left-hand side and select **Pods**.
   3. Search for a Pod labeled with `app=istio-ingressgateway` and click on the Pod's name.
   ![Search for a Pod with `app=istio-ingressgateway` label](../../../assets/search-for-istio-ingress-gateway.svg)
   4. Scroll down to find the `Containers` section and check which ports the `istio-proxy` container uses.
   ![Check ports used by istio-proxy](../../../assets/check-istio-proxy-ports.svg)


3. If the ports `80` and `443` are not used, check the logs of the Istio Ingress Gateway container for errors related to certificates.

   <!-- tabs:start -->
   #### **kubectl**
   Run the following command:
   ```bash
   kubectl logs -n istio-system -l app=istio-ingressgateway -c ingress-sds
   ```
   
   #### **Kyma Dashboard**
   Click on the **View Logs** button.
   ![View logs of the istio-proxy-container](../../../assets/view-istio-proxy-logs.svg)

   <!-- tabs:end -->


4. In the case of certificate-related issues, make sure that the `kyma-gateway-certs` and `kyma-gateway-certs-cacert` Secrets are available in the `istio-system` Namespace and that they contain proper data.

   <!-- tabs:start -->
   #### **kubectl**
   Run:
   ```bash
    kubectl get secrets -n istio-system kyma-gateway-certs -oyaml
    kubectl get secrets -n istio-system kyma-gateway-certs-cacert -oyaml
   ```

   #### **Kyma Dashboard**

   ...

   <!-- tabs:end -->
    

5. To regenerate a corrupted certificate, follow the tutorial to [Set up or update a custom domain TLS certificate in Kyma](https://kyma-project.io/docs/kyma/latest/03-tutorials/00-security/sec-01-tls-certificates-security/). If you are running Kyma provisioned through Gardener, follow the [Gardener troubleshooting guide](https://kyma-project.io/docs/kyma/latest/04-operation-guides/troubleshooting/security/sec-01-certificates-gardener/) instead.

   >**NOTE**: Remember to switch back to the `distroless` image once you've resolved the issue.
