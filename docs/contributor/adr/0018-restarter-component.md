# Restarter component in Istio module

## Status

Accepted

## Context

This document proposes changes to Istio's restarter component to enhance its functionality and address existing
limitations.

This document describes the following technical points related to the implementation:

- Lifecycle of the component
- Features and capabilities of the component
- Predicate logic for workload restart
- Event handling and reconciliation process
- Metrics and observability

This document does NOT enforce the following aspects of the implementation:

- Using specific libraries or frameworks
- Code placement
- Design patterns

The libraries and code snippets are suggestions.
These final aspects are left to the discretion of the implementer, as long as they do not conflict with the requirements
and constraints outlined in this document.

### Problem

The current implementation of the restarter component has certain limitations that affect its reliability. These
limitations include:

- Exponential memory growth due to the number of API calls to the Kubernetes API server, which can lead to performance
  degradation.
- Execution during the Istio CR reconciliation, which can delay configuration changes and affect the overall
  responsiveness of the application.
- Lack of customizability for the customer to configure and control the restart behavior of workloads, which can lead to
  unexpected restarts and potential downtime.
- Code quality issues that make the restarter component difficult to maintain and extend, leading to increased technical
  debt and bugs.

### Background

The restarter component is a critical functionality of the Istio module that ensures the workloads running in the Istio
service mesh are up to date during updates.
The first implementation was introduced in early versions of the module and has been adjusted over time. However, recent
bug reports and investigations have identified the current implementation as one of the major causes of application
instability.
Restarts are handled in one big bang because the component works within a single reconciliation loop of the Istio CR.
Restarting everything in a single run can cause massive downtimes that the customer does not expect.
We have also identified that the current implementation causes the Istio module to have exponential memory growth on
large clusters. The root cause is the number of API calls to fetch the list of workloads running in the Istio service
mesh. To admit a workload for restart, be it a Deployment, StatefulSet or DaemonSet, the restarter works on a list of
uncorrelated pods.
In a single reconciliation loop, the restarter fetches the list of all pods in a cluster in paginated chunks of 100.
For each chunk, the restarter admits the pods to the predicates, which usually requires additional API calls to retrieve
the parent of an admitted pod during restart. Listing all pods in a cluster in a single reconcile loop, even with
pagination, costs a lot of memory. On larger clusters we might end up with a stale cache, which can cause unwanted API
errors and another restart run. With regular errors, the Istio module can end up in a loop of restarts, causing a lot of
downtime for the customer.

This pagination was introduced to avoid overloading the API server with an excessive number of calls. That introduced a
new problem: the paginated chunks use a cursor with a TTL of 5 minutes. If the restart takes longer than 5 minutes, for
example on large clusters, the cursor expires and the entire operation must start over.

We have had several reports from customers who were surprised by restarts that caused downtime for them. There is no way
for the customer to control the behavior of restarts.

Due to the code quality issues, there are many problems with code testability and maintenance. The current
implementation contains a lot of imperative code, which mutates uncontrollably and cannot be properly unit tested.

## Decision

We have decided to implement a new restarter component that addresses the limitations stated above. The new component
is designed to be more efficient, reliable, and customizable, while also improving code quality and maintainability.
The new restarter component is implemented as a separate deployment that runs independently of the Istio reconciliation
loop, but its lifecycle is managed by the Istio module.
The component is responsible for monitoring the workloads in the Istio service mesh and restarting them as needed, based
on the default configuration and overrides provided by the user.

The new restarter works on an asynchronous, event-driven model. It watches workloads, reacts to changes, and reconciles
them asynchronously, implementing the Kubernetes controller pattern. Additionally, it uses time-based reconciliation to
regularly ensure that workloads are up to date with the current state.

It fetches the list of Pods top-to-bottom. This updated behavior starts from the parent workload (Deployment,
StatefulSet, DaemonSet) and then uses selectors to retrieve all child Pods to admit.
This greatly reduces the number of API calls and the size of the objects to admit in a single loop. It also ensures that
the admitted workload is up to date with the current cache state.

### Lifecycle of the restarter component

To address the memory and performance issues, the new restarter component must be installed as a separate application.
The component installation and removal are still managed by the Istio module.
To handle installation and uninstallation of the many Kubernetes resources required by the component (Deployment,
ServiceAccount, RBAC, Service), we use Helm. The Istio module embeds a Helm chart and applies it programmatically using
the Helm Go SDK (`helm.sh/helm/v3`) directly from controller code — no external `helm` binary is required.

This lifecycle is heavily inspired by Gardener's extensions pattern, which handles the installation of an application
via embedded Helm charts.

The Helm chart is embedded at compile time using Go's `embed.FS`. The installation of the component is integrated into
the Istio CR reconciliation loop and implements the Helm lifecycle:

1. **Install/Upgrade**: When the Istio CR is present and the Istio mesh is installed successfully, the reconciler renders
   the Helm chart (using `helm/pkg/action.NewInstall` / `action.NewUpgrade`) and applies the resulting manifests via
   server-side apply.
2. **Uninstall**: When the Istio CR is deleted (detected via finalizer), the reconciler calls `action.NewUninstall` to
   tear down the Helm release. This automatically removes all chart-owned resources.
3. **Drift detection**: The reconciler compares the desired chart values derived from the Istio CR against the
   last-applied values stored in the Helm release secret. If they differ, an upgrade is triggered.

The Helm release is stored in the same namespace as the Istio CR (usually `kyma-system`) as a Secret with Helm labels.
This allows standard `helm list` introspection for debugging.

Since the Istio version is tightly coupled to the module version, the restarter component version is also coupled to the
Istio module version. This means that when the Istio module is updated, the restarter component is updated as well.

Configuration of the restarter component is provided via a `.spec.restarter` field in the Istio CR. The reconciler
renders the Helm chart with values derived from this field, and any changes to it trigger a Helm upgrade.
The configuration is optional, and the component has a default configuration that is applied when fields are not present.
The default configuration is defined in the Helm chart's `values.yaml` file, ensuring that the customer gets the full
functionality without any configuration, while also allowing them to customize the behavior of the component if they
want to.

```yaml
spec:
  restarter:
    enabled: true
    logLevel: info
    reconcileInterval: 15m
    deployment:
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: 200m
          memory: 256Mi
```

Changing the lifecycle of the component also requires adjusting the decisions made
in [ADR-0002](./0002-istio-cr-status-improvements.md) regarding the restart conditions.
This change **drops** the `ProxySidecarRestartSucceeded` condition and instead introduces a new condition
`RestarterComponentReady` that reflects the state of the restarter component. The new condition is set to `True` when
the component is installed and running successfully, and `False` when it is not.
The Istio module does not attempt to restart workloads if the restarter component is not ready, and logs a Processing
message instead.

If the restarter component fails to start after a certain time, or after a number of restarts, the Istio module sets the
`RestarterComponentReady` condition to `False`, applies a proper condition description, and logs an error.

From now on, the Istio module does not change the CR status based on the restart status of workloads, but only reflects
the state of the restarter component.

| CR state       | Type                             | Status    | Reason                                | Message                                                         |
|----------------|----------------------------------|-----------|---------------------------------------|-----------------------------------------------------------------|
| ~~Processing~~ | ~~ProxySidecarRestartSucceeded~~ | ~~True~~  | ~~ProxySidecarRestartSucceeded~~      | ~~Proxy sidecar restart succeeded~~                             |
| ~~Error~~      | ~~ProxySidecarRestartSucceeded~~ | ~~False~~ | ~~ProxySidecarRestartFailed~~         | ~~Proxy sidecar restart failed~~                                |
| ~~Processing~~ | ~~ProxySidecarRestartSucceeded~~ | ~~False~~ | ~~ProxySidecarPartiallySucceeded~~    | ~~Proxy sidecar restart partially succeeded~~                   |
| ~~Warning~~    | ~~ProxySidecarRestartSucceeded~~ | ~~False~~ | ~~ProxySidecarManualRestartRequired~~ | ~~Proxy sidecar manual restart is required for some workloads~~ |
| Ready          | RestarterComponentReady          | True      | RestarterComponentReady               | Restarter component is installed and running successfully       |
| Processing     | RestarterComponentReady          | False     | RestarterComponentNotReady            | Restarter component is not ready                                |

### Features and capabilities of the component

We have identified the following features and capabilities that the new restarter component must have.

#### Support for limited workload types

After analyzing the current implementation, we identified that support for Pods and Jobs was not implemented. Instead,
the restarter returned a Warning.
This behavior is misleading, as it gives the impression that restarting these workloads is supported, while in reality
they are not.

To further simplify the decision, the component supports restart only for workloads managed by a Deployment,
StatefulSet, DaemonSet or ReplicaSet — that is, workloads that support rolling restarts or self-healing.
To restart a workload, we use behavior similar to the `kubectl rollout restart` command, which updates the pod template
spec with a new annotation that triggers a rolling restart of the workload.
Every time a restart decision is made, the component updates the pod template spec with a new annotation
`restarter.istio-operator.kyma-project.io/restartedAt`.

ReplicaSets, however, do not support rolling restarts. To restart workloads covered by a ReplicaSet, the restarter
deletes the pods that belong to the ReplicaSet and relies on the ReplicaSet controller to create new ones.
This special feature only supports ReplicaSets that are **not** managed by a parent object.
This is a fragile approach, as it can cause downtime if the ReplicaSet does not have enough replicas to maintain
availability or is not covered by a PodDisruptionBudget. The user must be aware of this and take responsibility for the
risk of downtime.

As mentioned in the Decision section, the component goes top-to-bottom. The controller first collects the parent object
(Deployment, StatefulSet, DaemonSet, ReplicaSet) and then uses selectors to retrieve all child Pods to admit in a single
reconcile loop. Then each workload (Parent + children) is evaluated through a set of predicates in a loop. If predicate 
supports only one type of workload, it must implement a type assertion and skip the evaluation for unsupported workload
types.

If demand for support of other workload types arises, we can implement it in the future. However, we do not support
other workload types in the first implementation.

#### Maintenance window support

The component must support a maintenance window configuration that allows the user to specify a time range during which
restarts are allowed.
This helps to avoid unexpected restarts during critical business hours. The maintenance window is defined as an optional
annotation added by the user on the workload.
For easier configuration, we introduce a new annotation
`restarter.istio-operator.kyma-project.io/maintenance-window` that supports a syntax of `Day-of-week HH:MM-HH:MM`
in UTC.

It is parsed as two parts: `<day-range> <time-range>`. The day range supports:

- Single day: Sat
- Range: Sat-Sun
- Comma list: Sat,Sun (if you want non-contiguous days later)

This stays human-readable, requires no cron knowledge, and covers the vast majority of real maintenance window patterns.

Validation parses the annotation and rejects invalid syntax by emitting an appropriate Event.
The check at reconcile time becomes:

1. Parse the day range → is `time.Now().Weekday()` within it?
2. Parse the time range → is the current time-of-day within it?
3. If the maintenance window is smaller than `reconcileInterval`, emit a `RestartConfigInvalid` event and skip.
4. Both true → proceed with restart; otherwise skip and requeue with `reconcileInterval` until the window opens.

#### Restart exclusions

The component must support a configuration that allows the user to specify workloads that should be excluded from
restarts.
This helps to avoid restarts of critical workloads that cannot tolerate downtime. We introduce a new annotation
`restarter.istio-operator.kyma-project.io/exclude` that supports a boolean value of `true` or `false`. If the annotation
is set to `true`, the workload is excluded from restarts.
If the annotation is not present, the workload is included in restarts by default.

This mode excludes the workload from **restarts handled by component**, including Istio updates that mitigate a CVE in 
the proxy. This does **NOT** exclude the workload from restarts caused by other means, like node drains.
Using this annotation also implies that the user is aware of the risk of not restarting the workload and takes
responsibility for keeping it updated.

The module emits a `RestartRequired` event for excluded workloads that must be restarted, so the user can take action to
restart them manually.

#### Watching namespace and workload-level Istio injection changes

A workload must restart when its own sidecar-relevant state drifts **or** when the `istio-injection` label on its
namespace changes — enabling injection on a namespace must roll the workloads inside it. A controller watching only its
own kind sees the first case but not the second, since a namespace edit produces no event on the workload object.

controller-runtime handles this with a secondary watch that maps the changed object back to the workloads it affects.
Alongside the primary `For(&appsv1.Deployment{})` watch, the controller adds a `Watches` on `&corev1.Namespace{}` with
an `EnqueueRequestsFromMapFunc` that lists the workloads in the changed namespace and enqueues a reconcile request for
each. A `predicate` narrows namespace events to those where the injection label actually changed, so unrelated namespace
edits do not trigger churn.

```go
ctrl.NewControllerManagedBy(mgr).
    For(&appsv1.Deployment{}).
    Watches(
		&corev1.Namespace{},
        handler.EnqueueRequestsFromMapFunc(deploymentsInNamespace),
        builder.WithPredicates(injectionLabelChanged()),
    ).
Complete(r)
```

The reconcile logic itself is unchanged: whether triggered by a workload event, a namespace event, or the time-based
requeue, it re-evaluates the same rule chain against the current workload state. The namespace watch only controls
*when* a reconcile is enqueued, not *how* the decision is made.

#### Extending the configuration and future-proofing

The options above (maintenance window, restart exclusion) are defined at the workload level. If there is a need to 
define them at the namespace level, we can introduce it in the future as a separate custom resource and plan a migration
from workload-level annotations to a custom resource with label selectors.
This allows configuring restarter behavior for multiple workloads with a single configuration.

### Predicate logic for workload restart

To decide whether a workload should be restarted, the component uses a set of predicates that are evaluated in order.
The predicate system is an existing approach and, in the long run, was a good choice, as it allows adding new predicates
in the future without changing the existing code.

However, the current implementation suffers from unnecessary code complexity and bugs. To address this, a new, static
approach is implemented.

Each reconciliation loop initializes a static list of predicates. The list must be immutable and cannot be changed at
runtime.

Each predicate implements a single `Rule` interface with a single method `Evaluate()`.
A workload is a Kubernetes object that implements the `Object` interface.
Instead of returning a boolean, `Evaluate()` returns a tri-state `Decision`. A boolean can only express "restart" or
"don't care", which forces conditions that *forbid* a restart (such as a maintenance window or an exclusion annotation)
to be handled outside the predicate list. A third state lets a predicate veto the restart from within the same chain,
short-circuiting evaluation.

The three possible decisions are:

- **`Restart`** — the workload has drifted from its desired state and should be restarted.
- **`Continue`** — this predicate is indifferent; the decision is deferred to the remaining predicates. This is the
  neutral element of the default OR evaluation.
- **`Stop`** — this predicate forbids the restart. Evaluation stops immediately and the workload is **not** restarted,
  regardless of what any other predicate would decide.

```go
type Decision int

const (
    Continue Decision = iota
    Restart
    Stop
)

type Rule interface {
    Evaluate(ctx context.Context, obj Object) Decision
}
```

Predicates only evaluate the workload based on its current state and must not have any side effects, such as fetching a
resource from the API.
Any static information required for the evaluation must be passed to the predicate via its constructor **before**
triggering workload evaluation.

Request-scoped values that are only known at evaluation time — such as the reference clock used to test a maintenance
window — are passed through the `context.Context` argument of `Evaluate()`. This keeps time-dependent rules
deterministic and testable without giving them the ability to reach out to the API.

Maintenance windows and exclusion annotations are modeled as ordinary rules that return `Stop`. Placing them at the
front of the predicate list means a workload inside a maintenance window or explicitly excluded is never restarted, no
matter how many drift predicates would otherwise match.

The predicates must be extensible so that each predicate can be used in different reconciliation loops.
The list of available predicates must be defined in a single place within a reconcile loop, as a list of `Rule`
implementations.
This list cannot mutate, meaning predicates cannot be added or removed at runtime. This ensures that the predicates are
evaluated in a consistent order and that the behavior of the component is predictable.

#### Naming scheme

Rule names must describe *the condition being asserted*, not the action taken. The name states the condition that the
rule detects as **true**; the `Decision` it returns (`Restart` or `Stop`) determines the direction. A developer must be
able to understand what the rule checks from its name alone, without knowing which decision it maps to.

- **`<Subject>Changed`** — for rules that return `Restart` when the workload has drifted from its desired state. Read as
  "the subject changed, so the workload is stale". Examples: `CniModeChanged`, `ProxyConfigChanged`,
  `CompatibilityModeChanged`.

- Veto rules that return `Stop` follow the same principle: name the active condition that causes the veto, not the veto
  itself. Examples: `MaintenanceWindowActive` (a window is currently open) and `WorkloadExcluded` (the workload carries
  an exclusion annotation).

These forms keep the name focused on *why* rather than *what*. A rule name must not describe the restart itself (for
example, `RestartOnCni` or `DoProxyRestart` are discouraged) — the restart is implied by an `Restart` decision, and the
veto by a `Stop`.

#### Logical operations of predicates

Each rule is a self-contained unit that evaluates a single aspect of the workload.
Rules can be combined using logical operations to create more complex matching criteria.

By default, evaluation iterates over the list of defined rules and applies the following logic. This implies a logical
OR over `Restart`, with `Stop` acting as a short-circuit:

- a `Stop` from any rule stops evaluation immediately — the workload is **not** restarted;
- otherwise, the first `Restart` marks the workload for restart;
- if every rule returns `Continue`, the workload is left untouched.

The priority of predicates is defined by the actions they implement. When implementing a new predicate, the developer
must ensure that the predicate is added to the list in the correct order, so that it is evaluated in the right context.
The order of predicates is defined by the actions they implement:

- Predicates that return `Stop` must be evaluated first, as they veto the restart and short-circuit evaluation;
- Predicates that return `Restart` must be evaluated after the `Stop` predicates.

### Event handling and reconciliation process

For better stability of the restarter, the implementation follows the standard Kubernetes controller pattern.

#### Manager and controllers

Each supported workload kind (Deployment, StatefulSet, DaemonSet) has its own dedicated controller-runtime controller.
All controllers are registered in a single `manager.Manager` instance, sharing the same client, cache, and metrics
server.

The restarter component watches for events on the supported workload types (Deployment, StatefulSet, DaemonSet,
ReplicaSet). Each workload type is covered by a controller. Controllers define watches for the supported workloads. When
an event is received, the controller enqueues a reconcile request for the workload. The events are asynchronous, which
ensures that workloads are not admitted at the same time. Additionally, the restarter implements a time-based requeue
mechanism to ensure that workloads are regularly checked for drift, even if no events are received. This is done by
setting a `reconcileInterval` in the configuration, which triggers a reconcile request for each workload.

For mesh configuration changes, e.g. trusted proxies or Prometheus merge changes, to ensure that customers get the
workload updates as fast as possible, controllers watch for changes to the Istio CR and enqueue a reconcile request for
all workloads when the CR is updated.

To protect the API server from bursts of restart patches, each controller's workqueue is configured with a rate
limiter. controller-runtime requeues reconcile requests through a rate limiter; the piece relevant here is its
token-bucket limiter, which caps the queue's aggregate throughput. A token-bucket limiter takes a sustained rate `r`
(tokens per second) and a burst size `b` (bucket capacity), refilling the bucket at `r` tokens per second so the
sustained average never exceeds `r`. This ceiling applies per controller; a single shared limiter can be passed to all
controllers if a global cap across workload kinds is required.

To tackle memory issues, the manager configures a cache transform on the watched workload types.
controller-runtime's cache supports a `TransformFunc` that is applied to every object before it is stored in the
informer cache. By setting a `TransformFunc` on each `ByObject` entry (or globally via `DefaultTransform`), the
restarter can strip fields that are never used in predicate evaluation — such as `managedFields`, `ownerReferences`, and
the full pod spec — before the object is committed to memory.

The transform function retains only the fields required for predicate evaluation. If any of the predicates require
additional fields, they must be added to the transform function.

**controller-runtime** provides `cache.TransformStripManagedFields()` as a built-in helper that removes `managedFields`
alone, which is the single largest contributor to object size. For the restarter, a custom `TransformFunc` will be
registered per object kind via `cache.ByObject.Transform` to retain only the fields listed above, giving tighter memory
bounds than the built-in helper alone.

#### Kubernetes Events

The Kubernetes Events API provides an easy way to observe the actions performed on the workload component.
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

#### Testability chart

The new design is testable because each layer is a pure, stateless unit with dependencies injected at construction. The
table below maps each unit to how it is tested and the property that makes that possible.

| Unit                  | Test strategy                                                     | What makes it testable                                     |
|-----------------------|-------------------------------------------------------------------|------------------------------------------------------------|
| Individual `Rule`     | Table-driven unit test: given an `Object`, assert the `Decision`  | Pure function of its input; no API calls or side effects   |
| Time-dependent rule   | Inject a fixed clock via `context.Context`; assert `Restart`/`Stop` | Reference time is passed in, not read from the wall clock  |
| Rule chain evaluation | Unit test asserting short-circuit and OR semantics                | Deterministic over an immutable, ordered rule list         |
| Reconciler            | envtest against a real API server; assert restart patch / events  | Controller-runtime pattern; no bespoke pod-listing to mock |

The unit tier above covers the decision logic in isolation. Two higher tiers verify the wired-up component against real
infrastructure.

**Integration tests** run the full manager (controllers, watches, workqueue, cache transforms) against an `envtest`
control plane — a real `kube-apiserver` and `etcd`, no kubelet. They assert the reconciliation path end-to-end at the
controller level: mutate a watched field, then check the reconciler enqueues, evaluates the rule chain, and issues the
expected restart patch. They also cover what unit tests cannot reach — event emission, requeue/rate-limiting, and that
the cache transform retains every field the rules read. With no kubelet, pods are simulated, so this tier validates
control flow rather than real rollouts.

**E2E tests** deploy the restarter on a live cluster alongside a real Istio installation and validate the
customer-observed outcome: apply a config change, then assert targeted workloads are rolled and become ready while
excluded and maintenance-window workloads are untouched. This is the only tier that verifies real pod rollouts and the
full module lifecycle (install, upgrade, uninstall). Being slow and brittle, it covers critical customer journeys only.

### Metrics and observability

The restarter component exposes a Prometheus metrics endpoint (`/metrics`) served by the controller-runtime manager.
All custom metrics are prefixed `istio_restarter_`. In addition to the metrics below, the manager automatically exports
the standard controller-runtime metrics (`controller_runtime_reconcile_total`,
`controller_runtime_reconcile_errors_total`, `controller_runtime_reconcile_time_seconds`, `workqueue_depth`,
`workqueue_adds_total`, `workqueue_retries_total`), which already cover reconcile throughput, error rate, and queue
depth per controller — so the custom metrics below focus on restarter-specific decisions rather than duplicating those.

**Cardinality note:** labels are restricted to bounded-cardinality dimensions (`kind`, `reason`, `result`). Per-workload
identity (namespace/name) is intentionally *not* a label — on a large cluster that would produce unbounded time series
and reintroduce the memory problem this ADR set out to solve. Per-workload restart facts are surfaced through Kubernetes
Events instead (filterable by `reportingComponent=istio-restarter`).

| Metric                                             | Type      | Labels                                                        | Description                                                                                                                                                                                                                                                                       |
|----------------------------------------------------|-----------|---------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `istio_restarter_restarts_total`                   | Counter   | `kind`, `result` (`success` / `error`)                        | Restart attempts, split by outcome. Restart throughput and failure rate derive from this single metric.                                                                                                                                                                           |
| `istio_restarter_skipped_total`                    | Counter   | `reason` (`excluded`, `maintenance_window`, `invalid_window`) | Workloads skipped by the preflight check or exclusion. Mirrors the skip reasons one-to-one.                                                                                                                                                                                       |
| `istio_restarter_restart_pending`                  | Gauge     | `kind`                                                        | Workloads a rule returned `Restart` for but a `Stop` rule (exclusion or maintenance window) suppressed — a restart is *required* but not performed. Should trend to zero after a maintenance window opens; nonzero-stuck means blocked. Corresponds to the `RestartRequired` event. |
| `istio_restarter_rule_evaluation_duration_seconds` | Histogram | `kind`                                                        | Time to evaluate the rule chain for one workload. Detects rules that became expensive.                                                                                                                                                                                            |
| `istio_restarter_maintenance_window_active`        | Gauge     | —                                                             | `1` while *any* configured maintenance window is currently open, else `0`. A cluster-wide debugging signal for "why did nothing restart" — deliberately unlabeled to stay bounded-cardinality.                                                                                    |
| `istio_restarter_component_info`                   | Gauge     | `version`, `istio_version`                                    | Constant `1`; carries the restarter build version and the target Istio version as labels. Lets dashboards correlate restart activity with a specific Istio upgrade.                                                                                                               |

## Consequences

Accepting this decision resolves the core problems identified in the current implementation and introduces a set of
improvements that make the restarter component more reliable, maintainable, and customer-friendly.

- **Memory and performance**: The current implementation runs inside the Istio CR reconcile loop and lists all Pods
  cluster-wide on every cycle, bypassing the cache and growing it to gigabytes on large clusters. The new component runs
  as a separate process with its own informer cache and a `TransformFunc` that strips unused fields, bounding memory to
  the fields predicates actually read and removing the impact on the main controller entirely.
- **Restart behavior**: The current "big bang" restarts all matching workloads in a single pass, causing
  customer-visible downtime during upgrades. The new component reconciles per workload as events arrive — including
  namespace `istio-injection` label changes fanned out to the workloads they affect — and restarts via annotation
  patches, delegating the rollout to the built-in controllers so PodDisruptionBudgets are respected and mass eviction is
  eliminated.
- **Customer control**: The current implementation offers no way to influence restarts. The new component adds a
  per-workload exclude annotation and a maintenance-window annotation (a plain `Sat-Sun 00:00-04:00` string, no cron
  knowledge required), giving customers explicit opt-out and scheduling control.
- **Supported workload types**: The current restarter nominally handles Pods and Jobs but only warns, silently skipping
  them. The new component restricts support to Deployment, StatefulSet, and DaemonSet (rolling restart via annotation
  patch) plus unmanaged ReplicaSets (restarted by pod deletion, at the user's own downtime risk), removing a class of
  silent failures.
- **Predicate system**: The existing system carries two interface types (`SidecarProxyPredicate`,
  `IngressGatewayPredicate`) with inconsistent contracts and known bugs. The new implementation replaces both with a
  single stateless `Rule` interface returning a tri-state `Decision` (`Restart`/`Continue`/`Stop`). The `Stop` state lets
  vetoes — maintenance windows, exclusions — live in the same immutable, ordered rule list rather than as special-cased
  logic, and rules are shared across all workload-kind controllers.
- **Testability**: The current imperative code mutates shared state and cannot be unit-tested. The new design makes
  rules pure functions of their input, with time and other context injected at evaluation, so the decision logic is
  covered by fast unit tests, the wired-up controller by `envtest` integration tests, and real rollouts by E2E tests.
- **Observability**: The current implementation surfaces outcomes only through Istio CR status conditions, reset every
  reconcile. The new component drops `ProxySidecarRestartSucceeded` for a `RestarterComponentReady` condition reflecting
  component health, and adds per-workload Kubernetes Events and bounded-cardinality Prometheus metrics, giving operators
  and customers independent diagnostic signals without reading controller logs.
- **Operational complexity**: The standalone application adds a second binary and Helm chart to build, version, and
  clean up on uninstall (via the Istio CR finalizer), plus a second informer cache. These costs are bounded and
  predictable, and the Helm lifecycle keeps installation and removal auditable — an acceptable trade-off given the
  memory and reliability problems it resolves have caused production incidents.
