# Migration Guide for External Name Alias's Behavior Change

## Context

To align with Kubernetes behavior, Istio 1.21 changed the behavior of the Service of type ExternalName.
This Service is now treated as an alias of the Service that it points to.
Since this caused some issues for SAP BTP, Kyma runtime users, the Istio module introduced a new annotation, **disable-external-name-alias**, to disable the change.
However, this annotation has been deprecated. If you are using it, read this migration guide and update your resources' configuration accordingly.

## VirtualService

According to [Istio 1.21 Upgrade Notes](https://istio.io/latest/news/releases/1.21.x/announcing-1.21/upgrade-notes/), the destination of a VirtualService cannot point to the Service of type ExternalName.
The limitation exists in the traffic coming from the internal mesh clients.
Upgrade notes for version 1.21 describe a migration path by creating a ServiceEntry, although in currently released Istio versions it has no effect.
Routing to the Service of type ExternalName, which aliases another type of Service, will also not work.
Instead, it has to point to the actual host aliased by the Service of type ExternalName.

The following configuration will not work:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ext-svc
  namespace: default
spec:
  type: ExternalName
  externalName: httpbin.org
  
---

apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: vs-ext-svc
  namespace: default
spec:
  gateways:
    - mesh
  hosts:
    - "httpbin.<rest-of-the-domain>"
  http:
    - match:
        - uri:
            prefix: /
      route:
        - destination:
            host: ext-svc.default.svc.cluster.local
            port:
              number: 80

```

Use the following configuration instead:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: vs-ext-svc
  namespace: default
spec:
  gateways:
    - mesh
  hosts:
    - "httpbin.<rest-of-the-domain>"
  http:
    - match:
        - uri:
            prefix: /
      route:
        - destination:
            host: httpbin.org
            port:
              number: 80
```

Remember to test if the new configuration works as expected and fits all your needs.

## DestinationRule

DestinationRule cannot point to the Service of type ExternalName.
If you have a DestinationRule that points to a Service of type ExternalName, update the DestinationRule to point to the actual host.
To still have the possibility to apply some of the Istio features in the given situation, you can create a ServiceEntry for the external host.

The following configuration will not work:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ext-svc
  namespace: default
spec:
  type: ExternalName
  externalName: httpbin.org

---

apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: destination-rule
  namespace: default
spec:
  host: ext-svc.default.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
```

Use the following configuration instead:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: external-httpbin
  namespace: default
spec:
  hosts:
    - httpbin.org
  ports:
    - number: 80
      name: http
      protocol: HTTP
  resolution: DNS
  location: MESH_EXTERNAL

---

apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: destination-rule
  namespace: default
spec:
  host: httpbin.org
```

Remember to test if the new configuration works as expected and fits all your needs.

## Service ExternalName Ports

In the new behavior, Istio ignores the ports set in the Service of type ExternalName.
In case you rely on them in any way, you must update your configuration to align with the new behavior.
Since the solution heavily depends on the use case, it has to be adjusted to the particular case.
