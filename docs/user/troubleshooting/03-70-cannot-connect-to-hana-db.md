# Troubleshooting connection issues to Hana DB

## Symptom

An application cannot connect to a Hana DB instance.

## Troubleshooting

Istio module default configuration does not restrict outbound traffic. This means that the application should be able to connect to the Hana DB instance.
To troubleshoot what exactly is causing the connection issue, you can follow these steps:

1. Try to connect to the Hana DB instance from outside of the cluster:
    1. Download the Hana DB client for your OS from the [SAP Hana tools download page](https://tools.hana.ondemand.com/#hanatools).
    2. Unpack the downloaded archive.
    3. Install the Hana DB client.
    4. Connect to the Hana DB instance using the following command:
    ```bash
    hdbsql -n <HANA_DB_INSTANCE_ADDRESS> -u <HANA_DB_USER> -p <HANA_DB_PASSWORD>
    ```
    for example:
    ```bash
    hdbsql -n aaa.bbb.ccc.ddd:30015 -u myuser -p mypassword
    ```
    5. If the connection is successful and you can execute queries, the issue is not related to the Hana DB instance.
2. Try to connect to the Hana DB instance from inside of the cluster:
    1. Build a docker image with the Hana DB client installed. You can use the following Dockerfile:
    ```Dockerfile
    FROM eclipse-temurin:17
    WORKDIR /build
    COPY client.tar client.tar
    RUN tar -xvf client.tar
    RUN echo "/usr/local/bin" | ./client/hdbinst

    ENTRYPOINT ["sleep", "8000"]
    ```
    Download the Hana DB client for Linux x86 64-bit from the [SAP Hana tools download page](https://tools.hana.ondemand.com/#hanatools) and save it as `client.tar` in the same directory as the Dockerfile. Then run the following command to build the image:
    ```bash
    docker buildx build --platform=linux/amd64 -t hdbsql .
    ```
    2. Test your image by running the following command:
    ```bash
    docker run --entrypoint "hdbsql" hdbsql -v
    ```
    Example output:
    ```
    HDBSQL version 2.20.20.1712178305, the SAP HANA Database interactive terminal.
    Copyright 2000-2024 by SAP SE.
    ```
    3. Publish the image to a container registry.
    4. Run the image in the Kubernetes cluster:
    ```bash
    kubectl create deployment hdbsql --image=<PUBLISHED_IMAGE_NAME>
    ```
    5. Attach to the pod and try to connect to the Hana DB instance using the following command:
    ```bash
    hdbsql -n <HANA_DB_INSTANCE_ADDRESS> -u <HANA_DB_USER> -p <HANA_DB_PASSWORD>
    ```
    6. If the connection is successful and you can execute queries, the issue is not related to setup of the cluster.
    7. You might also want to check the connection from a pod that has Istio sidecar injected. In that case, create the deployment in a namespace with Istio sidecar injection enabled. The connection should still be successful.

