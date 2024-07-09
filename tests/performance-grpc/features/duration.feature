Feature: gRPC performance with a time duration boundary
  Scenario Outline: Run the test for duration=<duration> with concurrency=<concurrency>
    Given "Duration" is set to "<duration>"
    And "Concurrency" is set to "<concurrency>"
    When the gRPC performance test is executed
    Then the test should run successfully

    Examples:
      | concurrency | duration |
      | 1           | 60       |
