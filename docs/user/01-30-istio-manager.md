# Istio Controller parameters 

You can configure Istio Controller using various parameters. All options are listed in this document.

### Reconcile interval

By default, the Istio module is reconciled every 10 hours or whenever the custom resource is changed. You can adjust this interval by modifying the manager's parameters. For example, you can set the `--reconciliation-interval` parameter to `120s`.

### All configuration parameters

| Parameter                         | Description                                                                                                                                                                                                      | Default |
|-----------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| -failure-base-delay duration      | Indicates the failure base delay for the rate limiter.                                                                                                                                                           | 1s      |
| -failure-max-delay duration       | Indicates the maximum failure delay.                                                                                                                                                                             | 16m40s  |
| -health-probe-bind-address string | Specifies the address the probe endpoint binds to.                                                                                                                                                               | :8091   |
| -kubeconfig string                | Contains paths to the kubeconfig files.                                                                                                                                                                          |         |
| -leader-elect                     | Enable the leader election for controller manager. Enabling the election ensures there is only one active controller manager.                                                                                    |         |
| -metrics-bind-address             | Specifies the address the metric endpoint binds to.                                                                                                                                                              | :8090   |
| -rate-limiter-burst               | Indicates the burst value for the bucket rate limiter.                                                                                                                                                           | 200     |
| -rate-limiter-frequency           | Indicates the bucket rate limiter frequency, which signifies the number of events per second.                                                                                                                    | 30      |
| -reconciliation-interval          | Indicates the time-based reconciliation interval.                                                                                                                                                                | 10h0m0s |
| -zap-devel                        | Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)                                         | true    |
| -zap-encoder                      | Indicates the way of Zap log encoding. The value is either `json` or `console`.                                                                                                                                  |         |
| -zap-log-level                    | Indicates Zap Level used to configure the verbosity of logging. The value is either `debug`, `info`, `error`, or any integer value greater than 0, corresponding to custom debug levels of increasing verbosity. |         |
| -zap-stacktrace-level             | Determines Zap Level at and above which stacktraces are captured. The value is either `info`, `error`, or `panic`.                                                                                               |         |
| -zap-time-encoding                | Indicates the format for Zap time encoding. The value is either `epoch`, `millis`, `nano`, `iso8601`, `rfc3339`, or `rfc3339nano`.                                                                               | 'epoch' |
