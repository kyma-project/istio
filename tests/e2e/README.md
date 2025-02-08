# End-to-end tests

This directory contains definitions and implementations of end-to-end tests for istio operator.
These tests are for testing the connectivity of istio components to external services, implementing user scenarios.

## Running end-to-end tests

E2E tests run against cluster that is set up as active in your `KUBECONFIG` file. If you want to change the target,
export an environment vairable `KUBECONFIG` to specific kubeconfig file. Make sure you have admin permission on a
cluster.

To run the E2E tests, use `make test-e2e` command in project root directory.
You can also use `go test -run '^TestE2E.*' ./tests/e2e/...`directly, if you don't want to use make.

## Writing end-to-end tests

If you want to add another E2E test, there are several topics to follow.

Tests use lightweight testing framework included in the Go language.
For more information and reference, read the official Testing package documentation: https://pkg.go.dev/testing.

The tests are separate functions that expect that test cluster is configured with Istio manager installed without Istio CR applied.
Each function has to configure the cluster in runtime and then clean up the cluster back to initial state, removing all resources that were created.

For cleanup tasks, use default `t.Cleanup()` function from Testing package.

For helper functions use `t.Helper()` call. Each helper function should accept `t *testing.T` as argument.
Returning errors in this case is not really necessary. Just use `t.Error()` or `t.Fatal()` like with usual tests.

Avoid using global variables unless necessary. Do not import variables from other test packages, unless the external
package is a library.

For logging additional information, use default `t.Log()` and `t.Logf()` functions from Testing package.

Each of the test should be put in its own package. Keep the fixtures under `testdata` directory in the same package.

Put libraries used in tests under `pkg` directory, just as in standard Go project scheme.

Function body does not follow a strict methodology. Just keep it simple. Usually they are written as procedural functions.
If you need to implement more advanced logic, remember to not leak the implementation outside the package/function.

Start name of each function with prefix `TestE2E`.
When running `go test`, we set up pattern for running tests that only start with this prefix.

When you want to conditionally run specific test, use environment variables.
Those environment variables must be set in the environment before running `go test`. Do not set the variable in the test file.
For example, when the test should run only on gardener, add the simple if statement at the beginning of your test function:
```go
if os.Getenv("VARIABLE") == "true" {
	// for better visibility, add message in the function call
	t.Skip()
}
```

When you need to test multiple options in single configuration of a cluster, consider using table-driven tests.
For more information, follow official Go documentation: https://go.dev/wiki/TableDrivenTests.
