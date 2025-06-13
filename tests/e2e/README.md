# End-to-End Tests

This directory contains definitions and implementations of end-to-end tests for Istio Operator.
These tests verify the connectivity of Istio components to external services, implementing user scenarios.

## Running End-to-End Tests

E2E tests run against a cluster that is set up as active in your `KUBECONFIG` file. If you want to change the target,
export an environment vairable `KUBECONFIG` to specific kubeconfig file. Make sure you have admin permission on a
cluster.

To run the E2E tests, use the `make test-e2e-egress` command in the project's root directory.
You can also use `go test -run '^TestE2E.*' ./tests/e2e/...`directly, if you don't want to use make.

## Writing End-to-End Tests

If you want to add another E2E test, there are several topics to follow.

Tests use lightweight testing framework included in the Go language.
For more information and reference, read the official Testing package documentation: https://pkg.go.dev/testing.

The tests are separate functions that expect that the test cluster is configured with Istio manager installed without the Istio CR applied.
Each function has to configure the cluster in runtime and then clean up the cluster back to its initial state, removing all created resources.

For cleanup tasks, use the default `t.Cleanup()` function from the `Testing` package.

For helper functions, use the `t.Helper()` call. Each helper function should accept `t *testing.T` as an argument.
Returning errors in this case is not really necessary. Just use `t.Error()` or `t.Fatal()` like with usual tests.

Avoid using global variables unless necessary. Do not import variables from other test packages, unless the external
package is a library.

For logging additional information, use the default `t.Log()` and `t.Logf()` functions from the `Testing` package.

Each of the tests should be put in its own package. Keep the fixtures under the `testdata` directory in the same package.

Put libraries used in tests under the `pkg` directory, just as in the standard Go project scheme.

The function's body does not follow a strict methodology. Just keep it simple. Usually, they are written as procedural functions.
If you need to implement more advanced logic, remember not to leak the implementation outside the package/function.

Start each function's name with the prefix `TestE2E`.
When running `go test`, we set up a pattern for running tests that only start with this prefix.

When you want to conditionally run specific test, use environment variables.
Those environment variables must be set in the environment before running `go test`. Do not set the variable in the test file.
For example, when the test should run only on Gardener, add the simple `if` statement at the beginning of your test function:
```go
if os.Getenv("VARIABLE") == "true" {
	// for better visibility, add message in the function call
	t.Skip()
}
```

When you need to test multiple options in a single configuration of a cluster, consider using table-driven tests.
For more information, follow official Go documentation: https://go.dev/wiki/TableDrivenTests.
