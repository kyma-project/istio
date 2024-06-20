Feature: gRPC performance tests with constant RPS
  Scenario Outline: Run the test for requests=<requests> with RPS=<rps> and concurrency=<concurrency>
    Given "Requests" is set to "<requests>"
    And "RPS" is set to "<rps>"
    And "Concurrency" is set to "<concurrency>"
    When the gRPC performance test is executed
    Then the test should run successfully

    Examples:
      | requests  | rps   | concurrency    |
      | 100       | 10    | 10             |
      | 1000      | 100   | 10             |
      | 10000     | 500   | 10             |
      | 10000     | 1000  | 200            |
      | 20000     | 5000  | 1000           |
      | 100000    | 5000  | 1000           |
