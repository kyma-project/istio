# Istio controller manager parameters 

You can configure Istio controller manager with various parameters, all options are listed below.

### Reconcile interval

By default, Istio module is reconciled every 10 hours or when custom resource is changed. You can set this interval by changing manager params, for example: `--reconciliation-interval=120s`.

### All configuration parameters
```
-failure-base-delay duration
    Indicates the failure base delay for rate limiter. (default 1s)
-failure-max-delay duration
    Indicates the failure max delay. (default 16m40s)
-health-probe-bind-address string
    The address the probe endpoint binds to. (default ":8091")
-kubeconfig string
    Paths to a kubeconfig. Only required if out-of-cluster.
-leader-elect
    Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
-metrics-bind-address string
    The address the metric endpoint binds to. (default ":8090")
-rate-limiter-burst int
    Indicates the burst value for the bucket rate limiter. (default 200)
-rate-limiter-frequency int
    Indicates the bucket rate limiter frequency, signifying no. of events per second. (default 30)
-reconciliation-interval duration
    Indicates the time based reconciliation interval. (default 10h0m0s)
-zap-devel
    Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default true)
-zap-encoder value
    Zap log encoding (one of 'json' or 'console')
-zap-log-level value
    Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
-zap-stacktrace-level value
    Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
-zap-time-encoding value
    Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```