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

If the restart doesn't help, follow these steps:

1. List all the Pods of Istio Ingress Gateway:

   ```bash
   kubectl get pod -l app=istio-ingressgateway -n istio-system -o name
   ```
   
2. Replace `{ISTIO_INGRESS_GATEWAY_POD_NAME}` with a name of a listed Pod and check the ports that the `istio-proxy` container uses:

   ```bash
   kubectl get -n istio-system pod {ISTIO_INGRESS_GATEWAY_POD_NAME} -o jsonpath='{.spec.containers[*].ports[*].containerPort}'
   ```

   #### **Kyma Dashboard**
   1. Go to the `istio-system` namespace.
   2. Navigate to the **Workloads** section on the left-hand side and select **Pods**.
   3. Search for a Pod labeled with `app=istio-ingressgateway` and click on the Pod's name.
   ![Search for a Pod with `app=istio-ingressgateway` label](../../../assets/search-for-istio-ingress-gateway.svg)
   4. Scroll down to find the `Containers` section and check which ports the `istio-proxy` container uses.
   ![Check ports used by istio-proxy](../../../assets/check-istio-proxy-ports.svg)


2. If the ports `80` and `443` are not used, check the logs of the Istio Ingress Gateway container for errors related to certificates.

   <!-- tabs:start -->
   #### **kubectl**
   Run the following command:
   ```bash
   kubectl logs -n istio-system -l app=istio-ingressgateway -c istio-proxy
   ```
   
   #### **Kyma Dashboard**
   Click on the **View Logs** button.
   ![View logs of the istio-proxy-container](../../../assets/view-istio-proxy-logs.svg)

   <!-- tabs:end -->


4. To make sure that a corrupted certificate is regenerated, verify if the **spec.enableKymaGateway** field of your APIGateway custom resource is set to `true`. If you are running Kyma provisioned through Gardener, follow the [Gardener troubleshooting guide](https://kyma-project.io/docs/kyma/latest/04-operation-guides/troubleshooting/security/sec-01-certificates-gardener/) instead.