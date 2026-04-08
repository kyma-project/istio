# Uneven Traffic Distribution with Destination Rules

## Symptom

After applying an Istio `DestinationRule` with outlier detection, you observe uneven traffic distribution across Pods of a service.
Some Pods receive significantly more requests than others, even though no errors or ejections are occurring.

## Cause

Istio's **locality load balancing** is implicitly activated whenever `outlierDetection` is configured in a `DestinationRule`,
even if `localityLbSetting.enabled` is not explicitly set. When active, locality load balancing routes traffic preferentially
to endpoints in the same locality (region, zone, or sub-zone) as the source workload, using outlier detection results to
determine endpoint health and trigger failover between localities.
In clusters where Pods are spread across multiple zones or nodes with unequal locality label distribution, this can skew
the load distribution, causing certain Pods to receive a disproportionate share of traffic.

This behavior is not caused by circuit breaking or Pod ejection.
Even with zero errors, the locality-aware routing algorithm can produce an uneven spread of requests across healthy Pods.

## Solution

Disable locality load balancing in the `DestinationRule` by explicitly setting `loadBalancer.localityLbSetting.enabled` to `false`
while keeping the outlier detection configuration in place.

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

Apply the updated `DestinationRule` to the cluster:

```bash
kubectl apply -f destination-rule.yaml
```

After applying the change, verify that traffic is distributed evenly across all Pods by checking your observability tooling
(for example, Grafana dashboards or Prometheus metrics).

## Additional Information

- Disabling locality load balancing does not affect the outlier detection behavior. Outlier detection will continue to monitor endpoint health and eject
unhealthy endpoints as configured, but traffic will be distributed evenly across all healthy endpoints regardless of locality.
- If your deployment spans multiple zones, and you intentionally want zone-aware routing, consider fine-tuning the
  [locality weight distribution](https://istio.io/latest/docs/reference/config/networking/destination-rule/#LocalityLoadBalancerSetting-Distribute)
  instead of disabling locality load balancing entirely.
