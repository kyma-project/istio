# Support for Configuration of Load Balancer types on AWS

## Status

Proposed

## Context

As requested we need to to support cluster configuration on AWS for the type of the `istio-ingressgateway`'s load balancer. AWS supports [different load balancers](https://docs.aws.amazon.com/elasticloadbalancing/latest/userguide/how-elastic-load-balancing-works.html). Per default Istio deploys a `Service` for the `istio-ingressgateway` without specifying anything and [AWS LoadBalancer controller](https://github.com/kubernetes-sigs/aws-load-balancer-controller) applies a `classic` (aka ELB v1) load balancer on AWS side, which we will refer as type `elb`. ELB works at both layers 4 (TCP/Network) and 7 (HTTP/Application).

Since classic ELB has quite a few limitations (e.g. supports only IPv4), we would like to extend the configuration for selecting a Network Load Balancer (NLB), which we will refer as type `nlb`. A NLB works at layer 4 only and can handle both TCP and UDP, supports IPv6, as well as TCP connections encrypted with TLS. Its main feature is that it has a very high performance.

## Decision

We will introduce configuration for `awsLoadBalancerType` within the `spec.config` of the Istio CustomResource (CR). The API will allow to conditionally specify the type of the `istio-ingressgateway` LB. Two types will be supported: `elb` and `nlb`.

## Consequences

The Istio CR API will be extended with configuration for `awsLoadBalancerType`, including validation for the only supported values: `elb` and `nlb`.

This results in the following Go structure:

```go
// Config is the configuration for the Istio installation.
type Config struct {
	...

	// optional, if not specified it defaults to `elb` for existing Istio installation and to `nlb` for new Istio installation
	// +kubebuilder:validation:Enum=elb,nlb
	AWSLoadBalancerType *string `json:"awsLoadBalancerType,omitempty"`
}
```

An example Istio CR could look as follows:

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    awsLoadBalancerType: nlb
```

* When switching to NLB Istio Module operator should apply additionally the following `serviceAnnotations` with Istio Operator Manifest:
```yaml
serviceAnnotations:
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: instance
    service.beta.kubernetes.io/aws-load-balancer-type: external
```

* When switching to the classic ELB Istio Module operator should remove the above `serviceAnnotations`.

> No downtime is expected when changing the LoadBalancer type on Istio Module reconciliation and no changes to the Kyma domain DNS entries are expected.

## Default Values

If `spec.config.awsLoadBalancerType` is not configured it defaults to `elb` for existing Istio installation and to `nlb` for new Istio installation. This will be determined at runtime (reconcile) time.
