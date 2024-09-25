# Default Istio Configuration

Istio Operator provides baseline values for the Istio installation, which you can override in the Istio custom resource (CR). It applies the following changes to customize Istio:

- Istiod (Pilot) and Ingress Gateway components are enabled by default.
- Automatic sidecar injection is disabled by default.
- To enhance  security and performance, both [Istio control plane and data plane](https://istio.io/latest/docs/ops/deployment/architecture/) use distroless version of Istio images. Those images are not Debian-based and are slimmed down to reduce any potential attack surface and increase startup time. To learn more, see [Harden Docker Container Images](https://istio.io/latest/docs/ops/configuration/security/harden-docker-images/).
- Resource requests and limits for Istio sidecars are modified to best suit the needs of the evaluation and production profiles. // czy to w czymś pomaga
- [Mutual TLS (mTLS)](https://istio.io/docs/concepts/security/#mutual-tls-authentication) is enabled cluster-wide in the `STRICT` mode.
- Ingress Gateway is expanded to handle HTTPS requests on port `443`. It redirects HTTP requests to HTTPS on port `80`. //do api gateway
- The use of HTTP 1.0 is enabled in the outbound HTTP listeners by the **PILOT_HTTP10** flag set in the Istiod component environment variables.
- No Egress limitations are implemented - all applications deployed in the Kyma cluster can access outside resources without limitations.
- The CNI component is provided as a DaemonSet, meaning that one replica is present on every node of the target cluster.
- The self-signed CA certificate’s bit length is set to 4096 instead of the default 2048.

## Configuration Based on the Cluster Size

The configuration of Istio resources depends on the cluster capabilities. If your cluster has less than 5 total virtual CPU cores or its total memory capacity is less than 10 Gigabytes, the default setup for resources and autoscaling is lighter. If your cluster exceeds both of these thresholds, Istio is installed with the higher resource configuration.

### Default Resource Configuration for Smaller Clusters

| Component       | CPU Requests | CPU Limits | Memory Requests | Memory Limits |
|-----------------|--------------|------------|-----------------|---------------|
| Proxy           | 10 m         | 250 m      | 32 Mi           | 254 Mi        |
| Ingress Gateway | 10 m         | 1000 m     | 32 Mi           | 1024 Mi       |
| Pilot           | 50 m         | 1000 m     | 128 Mi          | 1024 Mi       |
| CNI             | 10 m         | 250 m      | 128 Mi          | 384 Mi        |

### Default Resource Configuration for Larger Clusters

| Component       | CPU Requests | CPU Limits | Memory Requests | Memory Limits |
|-----------------|--------------|------------|-----------------|---------------|
| Proxy           | 10m          | 1000m      | 192Mi           | 1024Mi        |
| Ingress Gateway | 100m         | 2000m      | 128Mi           | 1024Mi        |
| Pilot           | 100m         | 4000m      | 512Mi           | 2Gi           |
| CNI             | 100m         | 500m       | 512Mi           | 1024Mi        |

### Default Autoscaling Configuration for Smaller Clusters

| Component       | minReplicas | maxReplicas |
|-----------------|-------------|-------------|
| Ingress Gateway | 1           | 1           |
| Pilot           | 1           | 1           |

### Default Autoscaling Configuration for Larger Clusters

| Component       | minReplicas | maxReplicas |
|-----------------|--------------|--------------|
| Ingress Gateway | 3            | 10           |
| Pilot           | 2            | 5            |