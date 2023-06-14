# Istio module default resources and autoscaling configuration

Istio module provides baseline values for installation of Istio. Those values can be overridden with configuration in your Istio Custom Resource.

## Cluster based default configuration

Istio module will install istio with configuration that depends on the cluster capabilities. If your cluster total virtual CPU cores number is less than `4` or total memory capability of the cluster is less than `10` Gigabytes, the default setup for resources and autoscaling will be lighter. If your cluster exceeds both thresholds Istio will install with higher resource configuration.

### Bigger clusters default resource configuration

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

### Bigger clusters default autoscaling configuration

The autoscaling configuration of the Istio components is as follows:

| Component       | Min replicas | Max replicas |
|-----------------|--------------|--------------|
| Pilot           | 2            | 5            |
| Ingress Gateway | 3            | 10           |

### Smaller clusters default resource configuration

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

### Smaller clusters default autoscaling configuration

The autoscaling configuration of the Istio components is as follows:

| Component       | Min replicas | Max replicas |
|-----------------|--------------|--------------|
| Pilot           | 1            | 1            |
| Ingress Gateway | 1            | 3            |

### CNI autoscaling

The CNI component is provided as a DaemonSet, meaning that one replica is present on every node of the target cluster.