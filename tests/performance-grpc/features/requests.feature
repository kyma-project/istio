Feature: gRPC performance tests for specific number of requests
  Scenario Outline: Run the test for requests=<requests> with concurrency=<concurrency>
    Given "Requests" is set to "<requests>"
    And "Concurrency" is set to "<concurrency>"
    When the gRPC performance test is executed
    Then the test should run successfully

    Examples:
      | requests  | concurrency |
      | 1000      | 1           |
      | 10000     | 10          |
      | 10000     | 100         |