# Migrate from using of Elastic Load Balancer (ELB) to Network Load Balancer (NLB) for Istio module running on AWS

> [!WARNING]
>
> Switching the load balancer type may cause brief downtime for the Istio Ingress Gateway.
> Make sure to plan the migration process accordingly,
> in a maintenance window that minimizes the impact on the application availability.
> The migration process from ELB to NLB is irreversible.

## Introduction

Until the 1.15 version, the Istio module was using the Elastic Load Balancer (ELB) as the load balancer type for the Istio Ingress Gateway.
Starting from the 1.15.0 version, the Network Load Balancer (NLB) is used as the new default.
This change was made to improve the feature compatibility with the AWS environment,
as well as making the Istio module installation more uniform across different cloud providers.

To facilitate safe migration from ELB to NLB, 
the 1.15 version of the module creates the `elb-deprecated` ConfigMap in the `istio-system` namespace.
This ConfigMap safeguards against downtime happening during the upgrade process to 1.16,
making sure that the ELB is still used as the load balancer type for the Istio Ingress Gateway as long as the ConfigMap is present.

## Migration

To migrate from using ELB to NLB for the Istio module running on AWS, follow these steps:
- Make sure that you are using the 1.15 or later version of the module.
- Remove the `elb-deprecated` ConfigMap from the `istio-system` namespace.
- The module will automatically switch to using the NLB as the load balancer type for the Istio Ingress Gateway.
