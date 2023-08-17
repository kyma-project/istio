---
title: Incompatible Istio sidecar version after the Istio module upgrade
---

## Symptom

You upgraded the Istio module and mesh connectivity is broken.

## Cause

By default, the Istio module has sidecar injection disabled - there is no automatic sidecar injection into any Pod in a cluster. For more information, read the document about [enabling Istio sidecar proxy injection](./01-60-enable-sidecar-injection.md).

The sidecar version in Pods must match the installed Istio version. Otherwise, mesh connectivity may be broken.
This issue may appear during the Istio module upgrade. When it is upgraded to a new version along with a new Istio version, existing sidecars injected into Pods remain in an original version.
The Istio module contains the `ProxySidecarReconcilation` component that performs a rollout for most common workload types, such as Deployments, DaemonSets, etc. The job ensures all running sidecars are properly updated.
However, some user-defined workloads can't be rolled out automatically. This applies, for example, to a standalone Pod without any backing management mechanism, such as a ReplicaSet or a Job.
Such user-defined workloads must be manually restarted to work correctly with the updated Istio version.

## Remedy

To learn if any Pods or workloads require a manual restart, follow these steps:

1. Check the installed Istio version using this method:

* From the `istiod` deployment in a running cluster, run:

   ```bash
   export KYMA_ISTIO_VERSION=$(kubectl get deployment istiod -n istio-system -o json | jq '.spec.template.spec.containers | .[].image' | sed 's/[^:"]*[:]//' | sed 's/["]//g')
2. Get the list of objects which require rollout. Find all Pods with outdated sidecars. The returned list follows the `name/namespace` format. The empty output means that there is no Pod that requires migration. To find all outdated Pods, run:

     <!--The command in step 2 can change once we start using solo.io images.-->

   ```bash
   COMMON_ISTIO_PROXY_IMAGE_PREFIX="europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2"
   kubectl get pods -A -o json | jq -rc '.items | .[] | select(.spec.containers[].image | startswith("'"${COMMON_ISTIO_PROXY_IMAGE_PREFIX}"'") and (endswith("'"${KYMA_ISTIO_VERSION}"'") | not))  | "\(.metadata.name)/\(.metadata.namespace)"'
   ```

3. After you find a set of objects that require the manual update, restart their related workloads so that new Istio sidecars are injected into the Pods.
