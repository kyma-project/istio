# gRPC Exposure Performance Test

This test is designed to measure the performance of gRPC exposure over VirtualService with the default configuration for the Istio module.

## Prerequisites

Istio module is installed and running in the cluster.
To provide a valuable test result, the Istio module configuration should ensure that the Istio Ingress Gateway has a constant number of replicas, and is not scaled up or down during the test.

## Running the Test on AWS

Out of the box, the test does not support running on AWS, as the `deploy-helm` Makefile target gets the external IP of the Istio Ingress Gateway.
The external IP is used in the `grpc-loadtest` command to send requests to the service.
The external IP is not available on AWS, so the test will fail.
To run the test on AWS, you must deploy the helm chart with the external address that AWS provides on the LoadBalancer service.

## Running the Test

1. Install the helm chart using the following command:

    ```bash
    make deploy-helm
    ```

2. Run the test using the following command:

    ```bash
    make grpc-load-test
    ```

3. To get the test's results, run the following command:

    ```bash
    make export-results
    ```
   The results are stored in the `results` directory in an HTML format.

## Test Configuration

[Feature files](./features) allow configuration of command line arguments for [grpc-loadtest](https://github.com/kyma-project/networking-dev-tools/tree/main/grpc-loadtest) tool. 
Each scenario is deployed as a separate Job. The job template is stored in [`/steps/job.yaml`](./steps/job.yaml).
