# Migration guide for external name alias behavior change

## Context

Istio with 1.21 version has changed the behavior of the Service of type ExternalName.
Previously, for the Service of type ExternalName a separate `cluster` in the Envoy config was created.
With the change, the Service of type ExternalName is treated as an alias of the Service that it points to.
This change has been introduced to align the behavior of Istio with handling of the Service of type ExternalName with the Kubernetes behavior.
Since it caused some issues for our users, we introduced a new annotation "disable-external-name-alias" to disable this behavior.
Under the hood, we apply `ENABLE_EXTERNAL_NAME_ALIAS` environment variable on the Istiod component.
At certain point Istio will drop the support for this environment variable, making the behavior change obligatory.
Currently, it's already removed from the main branch with the [PR](https://github.com/istio/istio/pull/52317).
We expect it to be a part of the 1.24 release.
Due to that, deprecation on the annotation will be introduced.
If you are affected by this change, you should update your configuration to align with the new behavior.

## Virtual Service

Due to the current Istio misbehavior, VirtualService cannot point with its destination to the Service of type ExternalName as the upgrade notes states [Ref](https://istio.io/latest/news/releases/1.21.x/announcing-1.21/upgrade-notes/).
The limitation exists in the traffic coming from the internal mesh clients.
Creating a ServiceEntry, has no effect despite upgrade notes for 1.21.
Routing to the Service of type ExternalName which alias other type of Service also will not work.
Instead, it has to point to the actual host that is being aliased by the Service of type ExternalName.
With [PR](https://github.com/istio/istio/pull/52589) merged, VirtualService should be able to point to the Service of type ExternalName for the internal mesh traffic.

The following will not work:

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

Do following instead:

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
If you have a DestinationRule that points to a Service of type ExternalName, you need to update the DestinationRule to point to the actual host.
To still have possibility to apply some of the Istio features in the given situation, you can create a ServiceEntry for the external host.

Following will no longer work:

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

Do following instead:

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

## Service ExternalName ports

In the new behavior ports set in the Service of type ExternalName are ignored by Istio.
In case if you rely on them in any way you need to update your configuration to align with the new behavior.
