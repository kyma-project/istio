#!/bin/bash
# Script to run zero downtime tests by executing the godog upgrade integration test and sending requests to
# the url exposed by a Virtual Service.
#
# The following process is executed:
# 1. Start the zero downtime requests in the background. The requests will be sent once the
# exposed host is reachable. The requests will be sent in a loop until the Virtual Service is deleted.
#  - Wait for 1 min until the host in the Virtual Service is available
#  - Send requests in parallel to the exposed host until the requests fail and in this case check if the Virtual Service
#    still exists to determine if the test failed or succeeded.
# 2. Run the godog upgrade test
# 3. Check if the zero downtime requests were successful.
set -eou pipefail

# The following trap is useful when breaking the script (ctrl+c), so it stops also background jobs
trap 'kill $(jobs -p)' INT

PARALLEL_REQUESTS=5


run_zero_downtime_requests() {

  # Get the host set in the APIRule
  exposed_host=$(kubectl get virtualservices upgrade-test-vs -n default -o jsonpath='{.spec.hosts[0]}')
  local url_under_test="https://$exposed_host/headers"

  # Wait until the host in the Virtual Service is available. This may take a very long time because the httpbin application
  # used in the integration tests takes a very long time to start successfully processing requests, even though it is
  # already ready.
  wait_for_url "$url_under_test"

  echo "zero-downtime: Sending requests to $url_under_test"

  # Run the send_requests function in parallel processes
  for (( i = 0; i < PARALLEL_REQUESTS; i++ )); do
    send_requests "$url_under_test" &
    request_pids[$i]=$!
  done

  # Wait for all send_requests processes to finish or fail fast if one of them fails
  for pid in ${request_pids[*]}; do
    wait $pid && request_runner_exit_code=$? || request_runner_exit_code=$?
    if [ $request_runner_exit_code -ne 0 ]; then
        echo "zero-downtime: A sending requests subprocess failed with a non-zero exit status."
        exit 1
    fi
  done

  exit 0
}

wait_for_url() {
  local url="$1"
  local attempts=1

  echo "zero-downtime: Waiting for URL '$url' to be available"

  # Wait for 1min
  while [[ $attempts -le 60 ]] ; do
    response=$(curl -sk -o /dev/null -L -w "%{http_code}" "$url")
  	if [ "$response" == "200" ]; then
      echo "zero-downtime: $url is available for requests"
  	  return 0
    fi
  	sleep 1
    ((attempts = attempts + 1))
  done

  echo "zero-downtime: $url is not available for requests"
  exit 1
}

# Function to send requests to a given url with optional bearer token
send_requests() {
  local url="$1"
  local request_count=0

  while true; do
    response=$(curl -sk -o /dev/null -w "%{http_code}" "$url")
    ((request_count = request_count + 1))

    if [ "$response" != "200" ]; then
      # If there is an error and the Virtual Service still exists, the test is failed, but if an error is received only when the
      # Virtual Service is deleted, the test is successful, because without an Virtual Service the request must fail as no host
      # is exposed. This was the most reliable way to detect when to stop the requests, since only sending requests
      # when the Virtual Service exists led to flaky results.
      if kubectl get virtualservices -A --ignore-not-found | grep -q .; then
        echo "zero-downtime: Test failed after $request_count requests. Canceling requests because of HTTP status code $response"
        exit 1
      else
        echo "zero-downtime: Test successful after $request_count requests. Stopping requests because Virtual Service is deleted."
        exit 0
      fi
    fi
  done
}

start() {
  # Start the requests in the background
  run_zero_downtime_requests &
  zero_downtime_requests_pid=$!

  echo "zero-downtime: Starting integration test scenario"

  go test -timeout 15m ./tests/integration -v -race -run "TestUpgrade" && test_exit_code=$? || test_exit_code=$?
  if [ $test_exit_code -ne 0 ]; then
    echo "zero-downtime: Test execution failed"
    return 1
  fi

  wait $zero_downtime_requests_pid && zero_downtime_exit_code=$? || zero_downtime_exit_code=$?
  if [ $zero_downtime_exit_code -ne 0 ]; then
    echo "zero-downtime: Requests returned a non-zero exit status, that means requests failed or returned a status not equal 200"
    return 2
  fi

  echo "zero-downtime: Test completed successfully"
  return 0
}

start && start_exit_code="$?" || start_exit_code="$?"
if [ "$start_exit_code" == "1" ]; then
  echo "zero-downtime: godog integration tests failed"
  exit 1
elif [ "$start_exit_code" == "2" ]; then
  echo "zero-downtime: Zero-downtime requests failed"
  exit 2
fi

echo "zero-downtime: Tests successful"
exit 0
