# Uneven Traffic Distribution with DestinationRules

## Symptom

After applying an Istio DestinationRule with outlier detection, you observe uneven traffic distribution across the Service Pods.
Some Pods receive significantly more requests than others, even though no errors or ejections occur.

## Cause

Istio implicitly activates locality load balancing whenever you configure **outlierDetection** in a DestinationRule,
even if you don't explicitly set **localityLbSetting.enabled**. When active, locality load balancing routes traffic preferentially
to endpoints in the same locality (region, zone, or sub-zone) as the source workload, using outlier detection results to
determine endpoint health and trigger failover between localities.
In clusters where Pods are distributed across multiple zones or nodes with unequal locality label distribution, locality-aware routing can skew load distribution. As a result, some Pods receive a disproportionate share of traffic.

Circuit breaking and Pod ejection do not cause this behavior.
Even with zero errors, the locality-aware routing algorithm can produce an uneven distribution of requests across healthy Pods.

## Solution

Disable locality load balancing in the DestinationRule by explicitly setting **loadBalancer.localityLbSetting.enabled** to `false`. Keep the outlier detection configuration in place.

**Before (uneven distribution):**

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: service-destinationrule
spec:
  host: service
  trafficPolicy:
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 10s
      baseEjectionTime: 60s
      maxEjectionPercent: 50
```

**After (balanced distribution):**

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: service-destinationrule
spec:
  host: service
  trafficPolicy:
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 10s
      baseEjectionTime: 60s
      maxEjectionPercent: 50
    loadBalancer:
      localityLbSetting:
        enabled: false
```

Apply the updated DestinationRule to the cluster:

```bash
kubectl apply -f destination-rule.yaml
```

After applying the change, verify that traffic is distributed evenly across all Pods by checking your observability tooling
(for example, Grafana dashboards or Prometheus metrics).

## Additional Information

- Disabling locality load balancing doesn't affect the outlier detection behavior. Outlier detection continues to monitor endpoint health and eject unhealthy endpoints as configured, but traffic is distributed evenly across all healthy endpoints regardless of locality.
- If your deployment spans multiple zones, and you intentionally want zone-aware routing, consider fine-tuning the
  [locality weight distribution](https://istio.io/latest/docs/reference/config/networking/destination-rule/#LocalityLoadBalancerSetting-Distribute)
  instead of disabling locality load balancing entirely.
