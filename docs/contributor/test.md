

|<div style="width:220px">Parameter</div>|Type|Description|
|---|:---:|:---|
| `-failure-base-delay`|  duration  | Indicates the failure-based delay for the rate limiter. By default, it's set to 1s. |
| `-failure-max-delay duration` | duration | Indicates the maximum failure delay. By default, it's set to 16m40s. |
| `-health-probe-bind-address` | string | The address the probe endpoint binds to. By default, it's set to `:8091`. |
| `-kubeconfig` |  string | Paths to a kubeconfig file. Only required if out-of-cluster. |
| `-leader-elect` | - | Enable the leader election for controller manager. Enabling the clection ensures there is only one active controller manager. |
| `-metrics-bind-address` | string | The address the metric endpoint binds to. By default, it's set to `:8090` |
| `-rate-limiter-burst` |  int | Indicates the burst value for the bucket rate limiter. By default, it's set to 200. |
| `-rate-limiter-frequency` | int | Indicates the bucket rate limiter frequency, signifying number of events per second. By default, it's set to 30.|
| `-reconciliation-interval` | duration | Indicates the time-based reconciliation interval. By default, it's set to 10h0m0s. |
| `-zap-devel` | | Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default true) |
| `-zap-encoder` | value | Zap log encoding. The value is either `json` or `console`. |
| `-zap-log-level` | value| Zap Level to configure the verbosity of logging. The value is either `debug`, `info`, `error`, or any integer value greater than 0, which corresponds to custom debug levels of increasing verbosity. |
| `-zap-stacktrace-level` |  value |  Zap Level at and above which stacktraces are captured. The value is either `info`, `error`, `panic`.|
| `-zap-time-encoding` | value | Zap time encoding. The value is either `epoch`, `millis`, `nano`, `iso8601`, `rfc3339`, or `rfc3339nano`. By default, it's set to `epoch`. |