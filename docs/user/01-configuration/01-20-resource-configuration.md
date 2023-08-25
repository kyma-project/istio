# Default resources and autoscaling configuration for Istio Operator

Istio Operator provides baseline values for the Istio installation. Those values can be overridden with configuration in the Istio custom resource (CR).

## Cluster-based default configuration

Istio Controller installs Istio with a configuration that depends on the cluster capabilities. If your cluster has less than 5 total virtual CPU cores or its total memory capacity is less than 10 Gigabytes, the default setup for resources and autoscaling is lighter. If your cluster exceeds both of these thresholds, Istio is installed with the higher resource configuration.

### Default resource configuration for larger clusters

| Component       |          | CPU   | Memory |
|-----------------|----------|-------|--------|
| Proxy           | Limits   | 1000m | 1024Mi |
| Proxy           | Requests | 10m   | 192Mi  |
| Ingress Gateway | Limits   | 2000m | 1024Mi |
| Ingress Gateway | Requests | 100m  | 128Mi  |
| Pilot           | Limits   | 4000m | 2Gi    |
| Pilot           | Requests | 100m  | 512Mi  |
| CNI             | Limits   | 500m  | 1024Mi |
| CNI             | Requests | 100m  | 512Mi  |

### Default autoscaling configuration for larger clusters

| Component       | Min replicas | Max replicas |
|-----------------|--------------|--------------|
| Pilot           | 2            | 5            |
| Ingress Gateway | 3            | 10           |

### Default resource configuration for smaller clusters

| Component       |          | CPU   | Memory |
|-----------------|----------|-------|--------|
| Proxy           | Limits   | 250m  | 254Mi  |
| Proxy           | Requests | 10m   | 32Mi   |
| Ingress Gateway | Limits   | 1000m | 1024Mi |
| Ingress Gateway | Requests | 10m   | 32Mi   |
| Pilot           | Limits   | 1000m | 1024Mi |
| Pilot           | Requests | 50m   | 128Mi  |
| CNI             | Limits   | 250m  | 384Mi  |
| CNI             | Requests | 10m   | 128Mi  |

### Default autoscaling configuration for smaller clusters

| Component       | Min replicas | Max replicas |
|-----------------|--------------|--------------|
| Pilot           | 1            | 1            |
| Ingress Gateway | 1            | 3            |

### CNI autoscaling

The CNI component is provided as a DaemonSet, meaning that one replica is present on every node of the target cluster.
