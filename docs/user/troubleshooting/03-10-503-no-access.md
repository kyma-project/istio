# Can't Access a Kyma Endpoint (503 status code)
<!-- open-source-only -->

## Symptom

You try to access a Kyma endpoint and receive the `503` status code.

## Cause

This behavior might be caused by a configuration error in Istio Ingress Gateway. As a result, the endpoint you call is not exposed.

## Solution

To fix this problem, restart the Pods of Istio Ingress Gateway.

1. List all available endpoints:

    ```bash
    kubectl get virtualservice --all-namespaces
    ```

2. To trigger the recreation of their configuration, delete the Pods of Istio Ingress Gateway:

     ```bash
     kubectl delete pod -l app=istio-ingressgateway -n istio-system
     ```

If the restart doesn't help, follow these steps:

#### **kubectl**

1. Check all ports used by Istio Ingress Gateway.

   1. List all the Pods of Istio Ingress Gateway:

      ```bash
      kubectl get pod -l app=istio-ingressgateway -n istio-system -o name
      ```

   2. For each of the listed Pods, replace `{ISTIO_INGRESS_GATEWAY_POD_NAME}` with the Pods'a name and check the ports that the `istio-proxy` container uses:

      ```bash
      kubectl get -n istio-system {ISTIO_INGRESS_GATEWAY_POD_NAME} -o jsonpath='{.spec.containers[*].ports[*].containerPort}'
      ```

2. If the ports `80` and `443` are not used, check the logs of the `istio-proxy` container for errors related to certificates.

   ```bash
   kubectl logs -n istio-system -l app=istio-ingressgateway -c istio-proxy
   ```

3. To make sure that a corrupted certificate is regenerated, verify if the **spec.enableKymaGateway** field of your APIGateway custom resource is set to `true`. If you are running Kyma provisioned through Gardener, follow the [Issues with Certificates on Gardener](https://kyma-project.io/04-operation-guides/troubleshooting/security/sec-01-certificates-gardener.html#issues-with-certificates-on-gardener) troubleshooting guide instead.

#### **Kyma Dashboard**

1. Check all ports used by Istio Ingress Gateway:

   1. Go to the `istio-system` namespace.
   2. In the **Workloads** section, select **Pods**.
   3. Search for a Pod labeled with `app=istio-ingressgateway` and click on its name.
   4. Scroll down to find the `Containers` section and check which ports the `istio-proxy` container uses.

2. If the ports `80` and `443` are not used, check the logs of the `istio-proxy` container for errors related to certificates. To do this, click **View Logs**.

3. To make sure that a corrupted certificate is regenerated, verify if the **spec.enableKymaGateway** field of your APIGateway custom resource is set to `true`. If you are running Kyma provisioned through Gardener, follow the [Issues with Certificates on Gardener](https://kyma-project.io/04-operation-guides/troubleshooting/security/sec-01-certificates-gardener.html#issues-with-certificates-on-gardener) troubleshooting guide instead.

<!-- tabs:end -->
