# Configure Istio CA with Custom Certificates
For enhanced security, replace Istio's default self-signed certificates with administrator-provided certificates.


## Context

By default, Istio generates its own self-signed root certificate and uses it to sign workload certificates. However, for production environments, you should use a proper Certificate Authority hierarchy for better security.

- Root CA: Runs offline on a secure machine
- Intermediate CAs: Issued by the Root CA to each Istio cluster
- Workload Certificates: Signed by the cluster's intermediate CA

For more information, see [Plug in CA Certificates](https://istio.io/latest/docs/tasks/security/cert-management/plugin-ca-cert/).

## Prerequisites


## Procedure

1. Obtain the root certificate and key //how //maybe it's a prereq

2. For each cluster, obtain an intermediate certificate and key for the Istio CA //how //maybe it's a prere

3. In each cluster, create a secret cacerts including all the input files `ca-cert.pem`, `ca-key.pem`, `root-cert.pem` and `cert-chain.pem`. For example, for cluster1:

    ```bash
    kubectl create secret generic cacerts -n istio-system \
        --from-file=cluster1/ca-cert.pem \
        --from-file=cluster1/ca-key.pem \
        --from-file=cluster1/root-cert.pem \
        --from-file=cluster1/cert-chain.pem
    ```