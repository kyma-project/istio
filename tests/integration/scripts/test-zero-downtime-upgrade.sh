#!/bin/bash
# Script to run zero downtime tests by executing the godog upgrade integration test and sending requests to
# the test application.
#
# The following process is executed:
# 1. Start the zero downtime requests in the background. The requests will be sent once the
# exposed host is reachable. The requests will be sent in a loop until the Virtual Service is deleted.
#  - Wait for 5 min until the Virtual Service exists
#  - If the test runs against a Gardener cluster, try to get the IP of the Gardener cluster for 1 min
#  - Wait for the exposed test application to be available for 2 min
#  - Send requests in parallel to the exposed host until the requests fail and in this case check if the Virtual Service
#    still exists to determine if the test failed or succeeded.
# 2. Run the godog upgrade test
# 3. Check if the zero downtime requests were successful.
set -eou pipefail

# The following trap is useful when breaking the script (ctrl+c), so it stops also background jobs
trap 'kill $(jobs -p)' INT

PARALLEL_REQUESTS=5


run_zero_downtime_requests() {

  wait_for_virtual_service_to_exist
  echo "zero-downtime: Virtual Service found"

  # Get the IP of the Gardener cluster if the shoot-info ConfigMap exists
  if kubectl get configmap shoot-info -n kube-system &> /dev/null; then
    ip=$(get_load_balancer_ip)
    if [ -z "$ip" ]; then
      echo "zero-downtime: Cannot get the IP of the Gardener cluster"
      exit 1
    fi
    host="$ip:80"
  else
    host="localhost:80"
  fi

  local url_under_test="http://$host/headers"

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

wait_for_virtual_service_to_exist() {
  local attempts=1
  echo "zero-downtime: Waiting for the Virtual Service to exist"
  # Wait for 5min
  while [[ $attempts -le 300 ]] ; do

    vs_crd=$(kubectl get crds virtualservices.networking.istio.io -A --ignore-not-found)
    if [ -z "$vs_crd" ]; then
      sleep 1
      ((attempts = attempts + 1))
      continue
    fi

    vs=$(kubectl get virtualservice -A --ignore-not-found) && kubectl_exit_code=$? || kubectl_exit_code=$?
    if [ $kubectl_exit_code -ne 0 ]; then
        echo "zero-downtime: kubectl failed when listing Virtual Services, exit code: $kubectl_exit_code"
        exit 2
    fi
  	[[ -n "$vs" ]] && return 0
  	sleep 1
    ((attempts = attempts + 1))
  done
  echo "zero-downtime: Virtual Service not found"
  exit 1
}

get_load_balancer_ip() {
  local namespace="istio-system"
  local service="istio-ingressgateway"
  local attempts=1
  local ip

  # Wait for 2 min
  while [[ $attempts -le 120 ]] ; do
    ip=$(kubectl get svc "$service" -n "$namespace" -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

    if [ -z "$ip" ]; then
      # If IP is not available, fallback to hostname and resolve it to IP
      hostname=$(kubectl get svc "$service" -n "$namespace" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
      if [ -n "$hostname" ]; then
        ip=$(dig +short "$hostname" | tail -n1)
      fi
    fi

    if [ -n "$ip" ]; then
      break
    fi
  	sleep 1
    ((attempts = attempts + 1))
  done

  echo "$ip"
}

wait_for_url() {
  local url="$1"
  local attempts=1

  echo "zero-downtime: Waiting for URL '$url' to be available"

  # Wait for 2 min
  while [[ $attempts -le 120 ]] ; do
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

# Function to send requests to a given url
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
