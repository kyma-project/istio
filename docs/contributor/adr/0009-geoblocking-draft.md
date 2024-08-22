# Geoblocking

## Status

Draft

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->

Geoblocking is a feature allowing to block incoming traffic from certain IP ranges, that are exclusive to certain countries or regions.
It can work against anonymous users and system to system network communication that are identified by its source IP address.
Since it's a network related concern, it was decided to implement it as a part of the Istio module.




## Decision
<!--- Explain the proposed change or action and the reason behind it. -->

### Considered architectural approaches

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