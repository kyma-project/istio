# Istio Controller RBAC configuration

Security is paramount, so we strictly follow the least privilege principle with the Istio Controller. While it needs permissions to manage Istio resources effectively, 
we carefully tailor them to specific tasks, avoiding unnecessary escalation to the level of all created resources.
As the Istio Controller orchestrates the deployment of Istio components, it necessitates comprehensive management privileges for Istio resources. 
These privileges must mirror the access control levels accorded to the resources themselves, ensuring seamless operation.

## Elevated permissions for Clusterroles
Istio's installation grants the `istiod-clusterrole-istio-system` broad permissions by using `*` verbs for accessing the `ingresses/status` resource in the `networking.k8s.io` API group.
The ClusterRole of the Istio controller therefore also requires broad cluster role permissions ('*').