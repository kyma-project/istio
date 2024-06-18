# gRPC exposure performance test

This test is designed to measure the performance of gRPC exposure over Virtual Service with the default configuration for the Istio module.

## Prerequisites

Istio module is installed and running in the cluster.

## Running the test

1. Install the helm chart with the following command:

    ```bash
    make deploy-helm
    ```

2. Run the test with the following command:

    ```bash
    make grpc-load-test
    ```

3. To get the tests results run the following command:

    ```bash
    make export-results
    ```
   
## Test configuration

[Feature files](./features) allow configuration of command line arguments for [grpc-loadtest](https://github.com/kyma-project/networking-dev-tools/tree/main/grpc-loadtest) tool. 
Each scenario is deployed as a separate Job. The job template is stored in [here](./steps/job.yaml).
