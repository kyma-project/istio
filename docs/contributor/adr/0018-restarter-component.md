# Restarter component in Istio module

## Status

Proposed

## Context

This document aims to introduce required changes in Istio's restarter component to enhance its functionality and address
existing limitations.

This document describes the following technical points related to the implementation:

- Lifecycle of the component
- Features and capabilities of the component
- Predicate logic for workload restart
- Event handling and reconciliation process
- Metrics and observability
- Available API

This document does NOT enforce the following aspects of the implementation:

- Using specific libraries or frameworks
- Code placement
- Design patterns

The libraries and code snippets are suggestions.
Those final aspects are left to the discretion of the implementer, as long as they do not conflict with the requirements
and constraints outlined in this document.

### Problem

Current implementation of the restarter component has certain limitations that affect its reliability. These limitations
include:

- Exponential memory growth due to the number of API calls to the Kubernetes API server, which can lead to performance
  degradation.
- Execution during the Istio CR reconciliation, which can cause delays in applying configuration changes and affect the
  overall responsiveness of the application.
- Lack of support for customizability for customer to configure and control the restart behavior of workloads, which can
  lead to unexpected restarts and potential downtime.
- Code quality issues that can make it difficult to maintain and extend the restarter component, leading to increased
  technical debt and bugs.

### Background

Restarter component is a critical functionality of istio module that ensures the workloads that are running in the Istio
service mesh are up to date during updates.
The first implementation has been introduced in early versions of the module and has been adjusted over time. However,
with the recent bug reports and investigations, it has been identified that the current implementation is one of the
major causes of application instability.
Restarts are handled in one big bang due to the fact that component works within single reconciliation loop of Istio CR.
Restarting everything in single run can cause massive downtimes which customer does not expect.
We have also identified that the current implementation causes the Istio module to have exponential memory growth on
large clusters. The root cause was number of API calls to fetch the list of workloads that are running in the Istio
service mesh.
Due to the fact that restarter and Istio CR reconcile loop work as a single binary, the API calls caused the controller
cache to grow uncontrollably, reaching gigabytes of allocated memory by the controller.
We had several reports from customers that they have been surprised by the restarts, and it has caused downtime for
them. There is no way for customer to control the behavior of restarts. This is a major concern and needs to be
addressed.

## Decision

We have decided to implement a new restarter component that will address the limitations stated above. The new component
will be designed to be more efficient, reliable, and customizable, while also improving code quality and
maintainability.
The new restarter component will be implemented as a separate application that will run independently of the Istio
reconciliation loop, but the lifecycle of the component will be managed by the Istio module.
The component will be responsible for monitoring the workloads in the Istio service mesh and restarting them as needed,
based on the default configuration and overrides provided by the user.

### Lifecycle of the component

To address the memory and performance issues, the new restarter component must be installed as a separate application.
The component installation and removal will still be managed by the Istio module.
To handle installation and uninstallation of many Kubernetes resources required by the component (Deployment,
ServiceAccount, RBAC, Service), we will use Helm. The Istio module will embed a Helm chart and apply it programmatically
using the Helm Go SDK (`helm.sh/helm/v3`) directly from controller code — no external `helm` binary is required.

This lifecycle is heavily inspired by the Gardener's extensions pattern, which handles the installation of application
via embedded helm charts.

The Helm chart will be embedded at compile time using Go's `embed.FS`. The installation of a component will be
integrated into the Istio CR reconciliation loop, and will implement the helm lifecycle:

1. **Install/Upgrade**: When the Istio CR is present and reconciled successfully, the reconciler renders the Helm
   chart (using `helm/pkg/action.NewInstall` / `action.NewUpgrade`) and applies the resulting manifests via server-side
   apply.
2. **Uninstall**: When the Istio CR is deleted (detected via finalizer), the reconciler calls `action.NewUninstall` to
   tear down the Helm release. This automatically removes all chart-owned resources.
3. **Drift detection**: The reconciler compares the desired chart values derived from the Istio CR against the
   last-applied values stored in the Helm release secret. If they differ, an upgrade is triggered.

The Helm release will be stored in the same namespace as the Istio CR (usually `kyma-system`) as a Secret with helm
labels. This allows standard `helm list` introspection for debugging.

Since Istio version is tightly coupled to the module version, the restarter component version will also be coupled to
the Istio module version. This means that when the Istio module is updated, the restarter component will also be
updated, causing subsequent requeue of all workloads by default.

Configuration of the restarter component will be provided via a `.spec.restarter` field in the Istio CR. The reconciler
will render the Helm chart with values derived from this field, and any changes to it will trigger a Helm upgrade.
The configuration will be optional, and the component will have a default configuration that is applied when fields are
not present. The default configuration will be defined in the Helm chart's `values.yaml` file, ensuring that customer
gets the full functionality without any configuration, but also allowing them to customize the behavior of the component
if they want to.

```yaml
spec:
  restarter:
    enabled: true
    logLevel: info
    reconcileInterval: 15m
    deployment:
      autoscaling:
        enabled: true
        minReplicas: 1
        maxReplicas: 3
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: 200m
          memory: 256Mi
```

Changing the lifecycle of the component also requires to adjust the decisions made
in [ADR-0002](./0002-istio-cr-status-improvements.md) regarding the restart conditions.
Introducing this change will drop usage of `ProxySidecarRestartSucceeded` condition, and instead introduce a new
condition `RestarterComponentReady` that will reflect the state of the restarter component. The new condition will be
set to `True` when the component is installed and running successfully, and `False` when it is not.
The Istio module will not attempt to restart workloads if the restarter component is not ready, and will log Processing
message instead.
If the restarter component failed to start after a certain time, or after a number of restarts, the Istio module will
set the `RestarterComponentReady` condition to `False`, and will log an Error message.
From now on, the Istio module will not change the CR status based on the restart status of workloads, but will only
reflect the state of the restarter component.

| CR state       | Type                             | Status    | Reason                                | Message                                                         |
|----------------|----------------------------------|-----------|---------------------------------------|-----------------------------------------------------------------|
| ~~Processing~~ | ~~ProxySidecarRestartSucceeded~~ | ~~True~~  | ~~ProxySidecarRestartSucceeded~~      | ~~Proxy sidecar restart succeeded~~                             |
| ~~Error~~      | ~~ProxySidecarRestartSucceeded~~ | ~~False~~ | ~~ProxySidecarRestartFailed~~         | ~~Proxy sidecar restart failed~~                                |
| ~~Processing~~ | ~~ProxySidecarRestartSucceeded~~ | ~~False~~ | ~~ProxySidecarPartiallySucceeded~~    | ~~Proxy sidecar restart partially succeeded~~                   |
| ~~Warning~~    | ~~ProxySidecarRestartSucceeded~~ | ~~False~~ | ~~ProxySidecarManualRestartRequired~~ | ~~Proxy sidecar manual restart is required for some workloads~~ |
| Ready          | RestarterComponentReady          | True      | RestarterComponentReady               | Restarter component is installed and running successfully       |
| Processing     | RestarterComponentReady          | False     | RestarterComponentNotReady            | Restarter component is not ready                                |
| Error          | RestarterComponentReady          | False     | RestarterComponentFailed              | Restarter component failed to start                             |

### Features and capabilities of the component

We have identified the following features and capabilities that the new restarter component must have.

#### Support for limited workload types

After the analysis of current implementation it's been identified that support for Pods and Jobs was not implemented.
Instead, the restarter was returning a Warning.
With ReplicaSet not owned by anything, we delete pod and hope replica controller would heal it.
This behavior is misleading, as it gives the impression that restarting these workloads is supported, while in reality
they are not.

To further simplify the decision, the component will support restart only for workloads that are managed by a
Deployment, StatefulSet or DaemonSet - that means workloads that support rolling restarts.
To restart workload we will use the similar behavior as `kubectl rollout restart` command, which is to update the pod
template spec with a new annotation that will trigger a rolling restart of the workload.
Every time the restart decision is made, the component will update the pod template spec with a new annotation
`restarter.istio-operator.kyma-project.io/restartedAt`.

If the demand for support of other workload types arises, we can implement it in the future. However, we will not
support workloads that do not support rolling restarts for now.

#### Maintenance window support

The component must support a maintenance window configuration that allows the user to specify a time range during which
restarts are allowed.
This will help to avoid unexpected restarts during critical business hours. The maintenance window will be defined as an
optional annotation added by the user on the workload.
For easier configuration, we introduce a new annotation
`restarter.alpha.istio-operator.kyma-project.io/maintenance-window` that supports a syntax of `Day-of-week HH:MM-HH:MM`
in UTC.

Parsed as three parts: <day-range> <time-range>. Day range supports:

- Single day: Sat
- Range: Sat-Sun
- Comma list: Sat,Sun (if you want non-contiguous days later)

This stays human-readable, requires no cron knowledge, and covers the vast majority of real maintenance window patterns.
Validation is a single regex:

```go
// "Sat-Sun 00:00-04:00"
var windowPattern = regexp.MustCompile(
`^(Mon|Tue|Wed|Thu|Fri|Sat|Sun)(?:-(Mon|Tue|Wed|Thu|Fri|Sat|Sun))? ([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$`,
)
```

The check at reconcile time becomes:

1. Parse day range → is time.Now().Weekday() within it?
2. Parse time range → is current time-of-day within it?
3. Both true → proceed with restart; otherwise skip and requeue with `reconcileInterval` until window opens.

#### Restart exclusions

The component must support a configuration that allows the user to specify workloads that should be excluded from
restarts.
This will help to avoid restarts of critical workloads that cannot tolerate downtime. We introduce a new annotation
`restarter.alpha.istio-operator.kyma-project.io/exclude` that supports a boolean value of `true` or `false`. If the
annotation is set to `true`, the workload will be excluded from restarts.
If the annotation is not present, the workload will be included in restarts by default.

This mode excludes workload from **all restarts**, including Istio critical CVE proxy updated.
Using this annotation also implies that the user is aware of the risks of not restarting the workload and take a
responsibility to keep it updated.

#### Extending the configuration and future-proofing

Those options are defined on workload-level. If there is a need to define them on namespace-level, we can introduce it
in the future as a separate custom resource and plan migration from workload-level annotations to custom resource with
label selectors.
This will allow configuring restarter behavior for multiple workloads with single configuration.

### Predicate logic for workload restart

To decide whether a workload should be restarted, the component uses a set of predicates that are evaluated in order.
Predicate system is an existing approach and in the long run was a good choice, as it allows to easily add new
predicates in the future without changing the existing code.

However, current implementation suffers from the unnecessary code complexity and bugs. To address this, a new, static
approach will be implemented.

Each predicate implements a single `Matcher` interface with a single method `Match()`.
Workload is a Kubernetes object that implements the `Object` interface. This interface shadows the `client.Object`
interface from controller-runtime.
The `Match()` method returns a boolean value indicating whether the workload matches the predicate.

`Match()` returns `true` when the state of the workload is different from the desired state, and the workload should be
restarted.

```go
type Object interface {
client.Object
}

type Matcher interface {
Match(obj Object) bool
}
```

Predicates only evaluate the workload based on its current state, and should not have any side effects, like fetching
resource from API.
Any additional information required for the evaluation should be passed to the predicate via its constructor **before**
triggering workload evaluation.

The predicates should be extensible in a way that each predicate can be used in different reconciliation loops.
The list of available predicates must be defined in a single place within a reconcile loop, as a list of `Matcher`
implementations.
This list cannot mutate, that means predicates cannot be added or removed at runtime. This is to ensure that the
predicates are evaluated in a consistent order and that the behavior of the component is predictable.

#### Naming scheme

Predicate names must describe *the condition that makes a restart necessary*, not the action taken — the action is
always the same (a restart). A predicate returns `true` when the workload has drifted from its desired state, so the
name
should read as an assertion of that drift. Two forms are used:

- **`<Subject>Changed`** — for predicates that detect a change relative to what the workload currently reflects. Read as
  "the subject changed, so the workload is stale". Examples: `CniModeChanged`, `ProxyConfigChanged`,
  `CompatibilityModeChanged`.

Both forms make the predicate list readable top-to-bottom as a set of restart triggers, and both keep the name focused
on
the *why* rather than the *what*. A predicate name should not describe the restart itself (for example, `RestartOnCni`
or
`DoProxyRestart` are discouraged) — the restart is implied by any predicate matching.

#### Logical operations of predicates

Each predicate is a self-contained unit that evaluates a single aspect of the workload.
Predicates can be combined using logical operations to create more complex matching criteria.

By default, evaluation of predicates iterates over a list of defined predicates. This implies the logical OR operation.
That means if any of the predicates returns `true`, the workload will be restarted.

When a workload must meet 2 or more conditions, `Matcher` implementation can be combined using helper function `And()`.

This function takes a list of `Matcher` implementations and returns a new `Matcher` that evaluates to `true` only if
all provided `Matcher` implementations evaluate to `true`.

```go
type andMatcher struct {
matchers []Matcher
}

func (a *andMatcher) Match(obj Object) bool {
for _, matcher := range a.matchers {
if !matcher.Match(obj) {
return false
}
}
return true
}

func And(matchers ...Matcher) Matcher {
return &andMatcher{matchers: matchers}
}
```

### Event handling and reconciliation process

For better stability of a restarter, the implementation would follow the standard Kubernetes controller pattern.

#### Manager and controllers

Each supported workload kind (Deployment, StatefulSet, DaemonSet) has its own dedicated controller-runtime controller.
All controllers are registered in a single `manager.Manager` instance, sharing the same client, cache, and metrics
server.

To tackle memory issues, the manager will configure a cache transform on the watched workload types.
controller-runtime's cache supports a `TransformFunc` that is applied to every object before it is stored in the
informer cache. By setting a `TransformFunc` on each `ByObject` entry (or globally via `DefaultTransform`), the
restarter can strip fields that are never used in predicate evaluation — such as `managedFields`, `ownerReferences`, and
the full pod spec — before the object is committed to memory.

The transform function will retain only the fields required for predicate evaluation. If any of the predicates require
additional fields, they must be added to the transform function.

**controller-runtime** provides `cache.TransformStripManagedFields()` as a built-in helper that removes `managedFields`
alone, which is the single largest contributor to object size. For the restarter, a custom `TransformFunc` will be
registered per object kind via `cache.ByObject.Transform` to retain only the fields listed above, giving tighter memory
bounds than the built-in helper alone.

#### Kubernetes Events

Kubernetes events API provides easy way to observe the actions performed on the workload component.
Events provide a way to track any actions made on the customer workload and can serve as a source of messages for
observability purposes.
Each controller holds a `record.EventRecorder` obtained at setup via `mgr.GetEventRecorderFor("istio-restarter")`. All
emitted Events share the same `reportingComponent` field, making them easy to filter:

```
kubectl get events --field-selector reportingComponent=istio-restarter
```

Events are emitted using `recorder.Event(obj, eventType, reason, message)` or `recorder.Eventf(...)` for formatted
messages. controller-runtime deduplicates events with the same reason and message within a short window, so a
high-frequency reconcile loop does not flood the Events API.

`RestartFailed` event is emitted when the restart operation itself fails, for example due to an API error while patching
the workload. Configuration problems such as a malformed maintenance window are reported through the dedicated
`RestartConfigInvalid` event and the `invalid_window` skip metric, not as a `RestartFailed` event.

`RestartRequired` event is emitted when a predicate would have matched but is suppressed by policy (for example, an
excluded workload that is running a stale proxy).

`RestartConfigInvalid` event is emitted by the preflight check when a workload's restarter configuration annotation is
malformed (for example, a maintenance window that does not match the expected syntax). It carries the workload identity
and the parse error so the user can locate and fix the annotation. It is deduplicated by Kubernetes, so a persistently
invalid annotation does not spam the Events API.

| Reason                 | Type    | Trigger                                          |
|------------------------|---------|--------------------------------------------------|
| `RestartSuccess`       | Normal  | Workload successfully restarted                  |
| `RestartFailed`        | Warning | Workload failed to restart                       |
| `RestartRequired`      | Warning | Workload requires manual restart                 |
| `RestartConfigInvalid` | Warning | Workload has a malformed restarter configuration |

#### Reconciliation decision diagram

```
Workload event (create / update / requeue)
           │
           ▼
    Fetch workload
    (NotFound → ignore)
           │
           ▼
  ┌──────────────────┐
  │ preflight check  │──deny──► skip + requeue, bump skipped_total{reason}
  │ (maint. window)  │          (RestartConfigInvalid event if malformed)
  └────────┬─────────┘
           │ allow
           ▼
  ┌──────────────────┐
  │ any Matcher      │──no──► no-op, no event
  │ returns true?    │
  └────────┬─────────┘
           │ yes (restart warranted)
           ▼
  ┌──────────────────┐         set restart_pending, bump skipped_total{reason=excluded},
  │ excluded?        │──yes──► emit RestartRequired event, do NOT restart
  └────────┬─────────┘
           │ no
           ▼
      do Restart()
           │
      ┌────┴────┐
      │ error?  │
      └────┬────┘
     yes   │    no
      ▼    │    ▼
  Warning  │  Normal Event
  Event    │  "Restarted"
  requeue  │  return OK
```

### Metrics and observability

The restarter component exposes a Prometheus metrics endpoint (`/metrics`) served by the controller-runtime manager.
All custom metrics are prefixed `istio_restarter_`. In addition to the metrics below, the manager automatically exports
the standard controller-runtime metrics (`controller_runtime_reconcile_total`,
`controller_runtime_reconcile_errors_total`,
`controller_runtime_reconcile_time_seconds`, `workqueue_depth`, `workqueue_adds_total`, `workqueue_retries_total`),
which
already cover reconcile throughput, error rate, and queue depth per controller — so the custom metrics below focus on
restarter-specific decisions rather than duplicating those.

**Cardinality note:** labels are restricted to bounded-cardinality dimensions (`kind`, `reason`, `result`). Per-workload
identity (namespace/name) is intentionally *not* a label — on a large cluster that would produce unbounded time series
and reintroduce the memory problem this ADR set out to solve. Per-workload restart facts are surfaced through Kubernetes
Events instead (filterable by `reportingComponent=istio-restarter`).

| Metric                                                  | Type      | Labels                                                        | Description                                                                                                                                                                                                                                                               |
|---------------------------------------------------------|-----------|---------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `istio_restarter_restarts_total`                        | Counter   | `kind`, `result` (`success` / `error`)                        | Restart attempts, split by outcome. Restart throughput and failure rate derive from this single metric.                                                                                                                                                                   |
| `istio_restarter_skipped_total`                         | Counter   | `reason` (`excluded`, `maintenance_window`, `invalid_window`) | Workloads skipped by the preflight check or exclusion. Mirrors the skip reasons one-to-one.                                                                                                                                                                               |
| `istio_restarter_restart_pending`                       | Gauge     | `kind`                                                        | Workloads a predicate matched but the exclusion check suppressed (a restart is *required* but not performed). Should trend to zero after a maintenance window opens; a stuck non-zero value means restarts are being blocked. Corresponds to the `RestartRequired` event. |
| `istio_restarter_predicate_evaluation_duration_seconds` | Histogram | `kind`                                                        | Time to evaluate the preflight check and predicate chain for one workload. Detects predicates that became expensive.                                                                                                                                                      |
| `istio_restarter_maintenance_window_active`             | Gauge     | —                                                             | `1` while *any* configured maintenance window is currently open, else `0`. A cluster-wide debugging signal for "why did nothing restart" — deliberately unlabeled to stay bounded-cardinality.                                                                            |
| `istio_restarter_component_info`                        | Gauge     | `version`, `istio_version`                                    | Constant `1`; carries the restarter build version and the target Istio version as labels. Lets dashboards correlate restart activity with a specific Istio upgrade.                                                                                                       |

## Consequences

Accepting this decision resolves the core problems identified in the current implementation and introduces a set of
improvements that make the restarter component more reliable, maintainable, and customer-friendly.

- **Memory and performance**: The current implementation executes entirely within the Istio CR reconciliation loop.
  Every reconcile cycle issues direct API calls to list all Pods across the cluster, bypassing the controller cache and
  causing the cache to grow uncontrollably — reaching gigabytes on large clusters. The new component runs as a separate
  process with its own informer cache and a `TransformFunc` that strips unused fields before objects are stored. This
  bounds memory usage to the minimum fields required for predicate evaluation and eliminates the impact on the main
  controller's memory footprint entirely.
- **Restart behavior**: The current implementation restarts all matching workloads in a single reconcile pass — a "big
  bang" approach that has caused customer-visible downtime during Istio upgrades. The new component replaces this with a
  controller-runtime reconcile loop per workload kind, where each workload is processed individually as an event
  arrives. Restarts are performed via annotation patches, which delegate the actual
  rollout to the Deployment/StatefulSet/DaemonSet controller and respect PodDisruptionBudgets. This eliminates the
  mass-eviction risk entirely.
- **Customer control**: The current implementation provides no mechanism for customers to influence when or whether
  their workloads are restarted. The new component introduces a per-workload exclude annotation and a maintenance window
  annotation, giving customers explicit control over both opt-out and scheduling of restarts. These are intentionally
  designed to require no Istio or cron expertise — a plain `Sat-Sun 00:00-04:00` string is sufficient.
- **Supported workload types**: The current restarter nominally handles Pods and Jobs but returns only a Warning for
  them, silently skipping restarts. Bare ReplicaSets are handled by deleting the pod and relying on the replication
  controller to heal, which is fragile and non-obvious. The new component explicitly restricts support to Deployment,
  StatefulSet, and DaemonSet — the only kinds that support rolling restarts — and rejects the ambiguity by design. This
  simplifies the predicate evaluation surface and removes a class of silent failures.
- **Predicate system**: The existing predicate system carries two interface types (`SidecarProxyPredicate` for pod-level
  evaluation and `IngressGatewayPredicate` for gateway evaluation) with inconsistent contracts and known bugs. The new
  implementation replaces both with a single `Matcher` interface that operates on a `client.Object`, is stateless, and
  receives all required context via its constructor. Predicates can be shared across all workload-kind controllers
  without modification, and the immutable matcher list prevents runtime state mutations that were a source of bugs in
  the prior implementation.
- **Observability**: The current implementation surfaces restart outcomes only through Istio CR status conditions (
  `ProxySidecarRestartSucceeded` and its variants), which provide a single aggregate view and are reset on every
  reconcile. The new component replaces these with a `RestarterComponentReady` condition that reflects the health of the
  component itself, and complements it with per-workload Kubernetes Events (filterable by
  `reportingComponent=istio-restarter`) and a set of Prometheus metrics covering throughput, errors, queue depth, and
  maintenance window state. This gives both operators and customers multiple independent signals to diagnose restart
  behavior without accessing controller logs.
- **Operational complexity**: The separation into a standalone application introduces new concerns: a second binary and
  Helm chart to build, publish, and version; Helm release state that must be cleaned up on uninstall via the Istio CR
  finalizer; and a separate informer cache that watches some of the same resource types as the main controller. These
  are real costs, but they are bounded and predictable. The Helm-based lifecycle provides a standard inventory of all
  owned resources, making installation and removal auditable. The trade-off is judged acceptable given that the memory
  and reliability problems it solves have directly caused production incidents.
