# Geoblocking

## Status

Draft

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->

Geoblocking is a feature allowing to block incoming traffic coming from certain IP addresses, that are exclusive to certain countries or regions.
Can be utilized against anonymous users and system to system network communication, where both are identified only by its source IP address.
Many companies are using geoblocking to protect their services from unwanted traffic, such as DDoS attacks, or to comply with legal regulations.
Therefore, creating this kind of feature is a convenience for the user, as it extracts the need to implement it on its own. 
It allows to utilize service mesh vision and capabilities through extracting a networking related concept outside the application, into the mesh.
Since it's a network related concern, it was decided to implement it as a part of the Istio module.

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
Istio allows to delegate the access control to an external authorization service, which may be just a regular Kubernetes service. This seems to be a simplest way to plug the geoblocking check of incomming connections.

### ip-auth service

For that purpose a new service 'ip-auth' is introduced. It has three main responsibilities:
- obtaining IP policies (allow lists, block lists)
- authorizing incomming connection requests
- informing about access decision (for auditing purposes)

![IP Auth](../../assets/ip-auth.svg)

### Modes of operation

The IP Auth service offers two modes of operation:
1. with static IP block list
2. with SAP geoblocking service (only SAP internal customers)

In the first mode the list of blocked IP ranges is read from a config map and stored in ip-auth application memory. The end-user may update the list of IP ranges at any time, so the IP-auth application is obliged to refresh it regularly. 

In the second mode the list of blocked IP ranges is received from the SAP geoblocking service. In order to connect to it the ip-auth requires a configmap with URLs and a secret with credentials. The list of blocked IP ranges is then in application memory and additionally in a configmap, which works as a persistent cache. This approach limits the number of list downloads and makes the whole solution more reliable if SAP geoblocking service is not available. The list of IP ranges should be refreshed once per hour.

In the second mode the ip-auth service reports the following events:
- policy list consumption (success, failure, unchanged)
- access decision (allow, deny)

![IP Auth modes](../../assets/ip-auth-modes.svg)

In order to ensure reliability and configurability a new Geoblocking Custom Resource and a new Geoblocking Controller is introduced. The controller is responsible for:
- managing the ip-auth service deployment
- managing authorization policy that plugs ip-auth authorizer to all incomming requests
- performing configuration checks (like external traffic policy)
- reporting geoblocking state

### Technical details

#### Policy download optimization

In order to reduce unnecessary updates of the list of blocked IP ranges the ip-auth service should use ETag mechanism, which is supported by the SAP geoblocking service.

#### Quick IP check

The list of blocked IP ranges may be big (thousands of entries), so analyzing the whole list with every incoming request will cause unnecessary delays.

Because the list of blocked IP ranges usually changes rarely, it is better to build a data structure that supports quick comparision of IP addresses. The good candidate is a radix tree.

#### IP Block list cache config map

Config map size can't exceed 1MB. Thus, the IP addresses must be stored in a short form, like just an array of strings. It can't store original json messages received from SAP geoblocking service.

#### Events retention

In order to not cause unnecessary delays the access events should be sent asynchronously:
- after making access decision the ip-auth should generate an event and store it in a memory queue
- separate thread should grab events from the queue and send them to the SAP geoblocking service.

This approach may cause issues if SAP geoblocking service responds slowly or does not respond at all. The ip-auth application must work reliably in this case and just drop events that can't be sent.

#### Headers used in check

The ip-auth service should take the following HTTP headers into consideration:
1. x-envoy-external-address - contains a single trusted IP address
2. x-forwarded-for - contains appendable list of IP addresses (modified by each compliant proxy)

IP-auth should block the connection request if any IP address in any of above headers belongs to any IP range in the block list.



## Considered architectural approaches

| Approach                                              | Pros                                                                             | Cons                                                                                                           |
|-------------------------------------------------------|----------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------|
| GB inside Istio CR                                    | Less boilerplate                                                                 | State of GBaaS shown in the IstioCR (spoils observability)                                                     |
|                                                       | Less devops                                                                      | Mess with IstioCR single purpose of mesh configuration and state reflection                                    |
|                                                       | Avoids too granular modules                                                      | Bug in logic related to GB kills also Istio reconciliation                                                     |
|                                                       |                                                                                  | Breaking separation of concerns between managed Istio installation and additional features on top of Istio     |
|                                                       |                                                                                  | Dependency to ip-auth and GBaaS in Istio Module                                                                |
|                                                       |                                                                                  | Affects complexity of Istio Module int/e2e tests                                                               |
|                                                       |                                                                                  | Blocker in Istio blocks also release of GB                                                                     |
|                                                       |                                                                                  | In the future GB might require more config, that would pollute IstioCR even more                               |
| ----------------------------------------------------- | -----------------------------------------------                                  | -------------------------------------------------------------------------------------------------------------- |
| GB inside Istio, separate CR                          | Slightly more boilerplate, but still not a lot                                   | Bug in logic related to GB kills also Istio reconciliation                                                     |
|                                                       | Less devops                                                                      | Breaking separation of concerns between managed Istio installation and additional features on top of Istio     |
|                                                       | Avoids too granular modules                                                      | Dependency to ip-auth and GBaaS in Istio Module                                                                |
|                                                       | Own resource to reflect config of GB                                             | Affects complexity of Istio Module int/e2e tests                                                               |
|                                                       | Own resource to reflect state of GB                                              | Blocker in Istio blocks also release of GB                                                                     |
| ----------------------------------------------------- | -----------------------------------------------                                  | -------------------------------------------------------------------------------------------------------------- |
| GB in own separate module                             | Separation of concerns                                                           | More devops                                                                                                    |
|                                                       | Resiliency compared to one module                                                | More boilerplate                                                                                               |
|                                                       | Own resource to reflect config of GB                                             | More modules granularity                                                                                       |
|                                                       | Own resource to reflect state of GB                                              | More MDs                                                                                                       |
|                                                       | Istio does not gain another dependency                                           |                                                                                                                |
|                                                       | We don't complicate Istio Module int/e2e tests                                   |                                                                                                                |
| ----------------------------------------------------- | -----------------------------------------------                                  | -------------------------------------------------------------------------------------------------------------- |
| GB in separate module with other 'mesh additions'     | Mesh installation module separated from features implemented on top of the mesh. | More devops                                                                                                    |
|                                                       | (rate limit, global rate limiting with redis etc.)                               | More boilerplate                                                                                               |
|                                                       | Avoid too granular modules (bundles more functionality)                          | More MD                                                                                                        |
|                                                       | Own resource to reflect config of GB                                             | Affects rate limiting by delaying it                                                                           |
|                                                       | Own resource to reflect state of GB                                              |                                                                                                                |
|                                                       | Resiliency compared to one module                                                |                                                                                                                |
|                                                       | Istio does not gain another dependency                                           |                                                                                                                |
|                                                       | We don't complicate Istio Module int/e2e tests                                   |                                                                                                                |

# API usage examples
```yaml
#INTERNAL
apiVersion: geoblocking.kyma-project.io/v1alpha1
kind: GeoBlocking
metadata:
  name: default
  namespace: kyma-system
spec:
  service:
    settingsConfigMap: "default/gb-settings"
    secret: "default/gb-secret"
---
#EXTERNAL
apiVersion: geoblocking.kyma-project.io/v1alpha1
kind: GeoBlocking
metadata:
    name: default
    namespace: kyma-system
spec:
  staticList:
    configMap: "default/ip-list"
```

## Consequences
<!--- Discuss the impact of this change, including what becomes easier or more complicated as a result. -->


## TODO:
Think where to put (CR / configmap / hardcoded  / etc.)
- ip-auth deployment settings
  - replicas
  - ip-auth image/version
  - memory / cpu limits and requests
- policy refresh interval
- events queue params (queue size, event TTL)
- protected URLs (does GB always protect all URLs?)
