## Performance Tests for Istio Module

These tests evaluate workloads with and without the Istio sidecar, using a virtual service. They are executed daily as part of the main workflow, and the results are available in the `summary-no-sidecar` and `summary-sidecar` reports.

Performance testing is conducted with the `k6` load testing tool, which simulates traffic to a service exposed via Istio. The tests measure the performance of HTTP GET and POST requests to a sample service configured to respond with headers and echo the request body.
Tests are performed on a service running in the Kyma cluster, both with and without Istio sidecar injection. This setup enables a direct comparison of performance metrics between the two configurations.

The `k6` tool is configured to run with 500 virtual users making constant requests for 1 minute. Tests are conducted on a Gardener AWS cluster, and the results are stored in HTML format.
You can run the tests using the `make test-performance` command, which deploys the necessary resources and executes the tests.

The HTTPBin service is deployed as a simple HTTP request and response service for testing HTTP clients. It is exposed via an Istio Virtual Service, which routes traffic to HTTPBin based on the request path.

The performance tests evaluate the following metrics:
- **http_req_failed**: The rate of failed HTTP requests, with a threshold of less than 1% failures.
- **http_req_duration**: The duration of HTTP requests, with a threshold that 95% of requests should complete in under 250ms.

Additionally, the Istio ingress gateway is configured with 10 replicas to handle increased traffic. This ensures the ingress gateway deployment always runs exactly 10 replicas, as both the minimum and maximum are set to 10.

## Performance of Deployments With and Without Istio Sidecar

Data collected from the performance tests is summarized in following tables, which compares the performance of deployments with and without Istio sidecar injection. The metrics include the number of successful and failed calls, data received and sent by the server, transfer speeds. Data presented in this table are an average of 2 values from 2 runs of the tests, so they are not exact values, but rather an average of the results. 


| Type       | No. of successful calls | No. of failed calls | Data received by server \[MB\] | Transfer speed (receiving) \[mB/s\] | Data sent by server \[MB\] | Transfer speed (sending) \[mB/s\] |
|------------|-------------------------|---------------------|-------------------------------|-------------------------------------|----------------------------|-----------------------------------|
| Sidecar    | 312444                  | 0                   | 291.78                        | 4.86                                | 24.35                      | 0.41                              |
| No-sidecar | 402166                  | 0                   | 553.67                        | 9.23                                | 31.26                      | 0.52                              |


The following table summarizes the performance of deployments with and without Istio sidecar injection, focusing on the median and 95th percentile of iteration durations for both configurations. The data is collected from two test runs on different dates.

| Date   | Sidecar - Median iteration duration [ms] | No-sidecar - Median iteration duration [ms] | Sidecar - 95th percentile [ms] | No-sidecar - 95th percentile [ms] |
|--------|-------------------------------------------|-------------------------------------|----------------------------------|-----------------------------------|
| 2025-07-08 | 84.66                                     | 67.83                               | 172.01                           | 139.31                             |
| 2025-07-09 |                    89.22                 |   66.37                             |     181.55                     |                 143.61            |


## Summary

These tests are sufficient, as there are many external dependencies on the platform that affect performance users should evaluate the performance of their own services with and without the Istio sidecar to observe any differences.
It is difficult to determine whether the results are good or bad for the platform as a whole, since performance depends on the specific service and its configuration.
The performance tests are intended to provide insights into the impact of Istio sidecar injection on service performance, helping developers make informed decisions about their service configurations in a Kyma environment.
