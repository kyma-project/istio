# Issues with Connection to SAP HANA Database

## Symptom

You're unable to connect an application to a SAP HANA Database instance.

## Troubleshooting

The Istio module's default configuration does not restrict outbound traffic. This means that the application should have no issues connecting to the SAP HANA Database instance.
To determine the cause of the connection issue, follow the troubleshooting steps.

### Connect to the SAP HANA Database Instance from Outside of the Cluster
1. Download SAP HANA Client for your operating system from the [SAP Development Tools](https://tools.hana.ondemand.com/#hanatools).
2. Unpack the downloaded archive.
3. Install SAP HANA Client.
4. Connect to SAP HANA Database instance using the following command:
    ```bash
    hdbsql -n {HANA_DB_INSTANCE_ADDRESS} -u {HANA_DB_USER} -p {HANA_DB_PASSWORD}
    ```
    For example:
    ```bash
    hdbsql -n aaa.bbb.ccc.ddd:30015 -u my_user -p mypassword
    ```
5. If the connection is successful and you can execute queries, the issue is not related to the SAP HANA Database instance.
### Connect to the SAP HANA Database Instance from Inside of the Cluster
1. Build a Docker image with the SAP HANA Client installed. You can use the following Dockerfile:
    ```Dockerfile
    FROM eclipse-temurin:17
    WORKDIR /build
    COPY client.tar client.tar
    RUN tar -xvf client.tar
    RUN echo "/usr/local/bin" | ./client/hdbinst

    ENTRYPOINT ["sleep", "8000"]
    ```
    Download the SAP HANA Client for Linux x86 64-bit from [SAP Development Tools](https://tools.hana.ondemand.com/#hanatools) and save it as `client.tar` in the same directory as the Dockerfile. Then, run the following command to build the image:
    ```bash
    docker buildx build --platform=linux/amd64 -t hdbsql .
    ```
2. To test your image, run the following command:
    ```bash
    docker run --entrypoint "hdbsql" hdbsql -v
    ```
    You get an output similar to this example:
    ```
    HDBSQL version 2.20.20.1712178305, the SAP HANA Database interactive terminal.
    Copyright 2000-2024 by SAP SE.
    ```
3. Publish the image to a container registry.
4. Run the image in the Kubernetes cluster:
    ```bash
    kubectl create deployment hdbsql --image={PUBLISHED_IMAGE_NAME}
    ```
5. Attach to the Pod and try to connect to the SAP HANA Database instance using the following command:
    ```bash
    hdbsql -n {HANA_DB_INSTANCE_ADDRESS} -u {HANA_DB_USER} -p {HANA_DB_PASSWORD}
    ```
6. If the connection is successful and you can execute queries, the issue is not related to the setup of the cluster.
7. Check the connection from a Pod that has the Istio sidecar injected. In that case, create the Deployment in a namespace with Istio sidecar injection enabled. The connection should be successful.

