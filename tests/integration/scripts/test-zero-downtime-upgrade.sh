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

VIRTUAL_SERVICES="upgrade-test-vs upgrade-test-vs-init-container"

# shellcheck disable=SC2120
run_zero_downtime_requests() {
  local vs_name=$1
  wait_for_virtual_service_to_exist "$vs_name"
  echo "zero-downtime: Virtual Service $vs_name found"

  local host_name
  host_name=$(kubectl get virtualservice "$vs_name" -o jsonpath='{.spec.hosts[0]}')
  echo "zero-downtime: Virtual Service $vs_name has host $host_name"

  # Get the IP of the Gardener cluster if the shoot-info ConfigMap exists
  if kubectl get configmap shoot-info -n kube-system &> /dev/null; then
    local ip=$(get_load_balancer_ip)
    if [ -z "$ip" ]; then
      echo "zero-downtime: Cannot get the IP of the Gardener cluster"
      exit 1
    fi
    local url_under_test="http://$ip:80/headers"
  else
    local url_under_test="http://localhost:80/headers"
  fi

  # Wait until the host in the Virtual Service is available. This may take a very long time because the httpbin application
  # used in the integration tests takes a very long time to start successfully processing requests, even though it is
  # already ready.
  wait_for_url "$url_under_test" "$host_name"

  echo "zero-downtime: Sending requests to $url_under_test for Virtual Service $vs_name and Host $host_name"

  # Run the send_requests function in parallel processes
  for (( i = 0; i < PARALLEL_REQUESTS; i++ )); do
    send_requests "$i" "$url_under_test" "$host_name" &
    pid=$!
    request_pids[$i]=$pid
    echo "zero-downtime: Started pid $pid for sending requests to $url_under_test for Virtual Service $vs_name and Host $host_name"
  done

  # Wait for all send_requests processes to finish or fail fast if one of them fails
  for pid in ${request_pids[*]}; do
    wait $pid && request_runner_exit_code=$? || request_runner_exit_code=$?
    if [ $request_runner_exit_code -ne 0 ]; then
        echo "zero-downtime: A sending requests subprocess with pid $pid failed with a non-zero exit status."
        exit 1
    fi
  done

  exit 0
}

# shellcheck disable=SC2120
wait_for_virtual_service_to_exist() {
  vs_name=$1
  local attempts=1
  echo "zero-downtime: Waiting for the Virtual Service $vs_name to exist"
  # Wait for 5min
  while [[ $attempts -le 300 ]] ; do

    vs_crd=$(kubectl get crds virtualservices.networking.istio.io -A --ignore-not-found)
    if [ -z "$vs_crd" ]; then
      sleep 1
      ((attempts = attempts + 1))
      continue
    fi

    kubectl get virtualservice "$vs_name" && kubectl_exit_code=$? || kubectl_exit_code=$?
    if [ "$kubectl_exit_code" -ne 0 ]; then
      echo "zero-downtime: kubectl failed when listing Virtual Service $vs_name, exit code: $kubectl_exit_code"
    else
      echo "zero-downtime: Virtual Service $vs_name exists"
      return 0
    fi

  	sleep 1
    ((attempts = attempts + 1))
  done
  echo "zero-downtime: Virtual Service $vs_name not found"
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
  local host_name="$2"
  local attempts=1

  echo "zero-downtime: Waiting for URL '$url' and host '$host_name' to be available"

  # Wait for 2 min
  while [[ $attempts -le 120 ]] ; do
    response=$(curl -sk -o /dev/null -L -w "%{http_code}" -H "Host: $host_name" "$url")
  	if [ "$response" == "200" ]; then
      echo "zero-downtime: $url and host $host_name is available for requests"
  	  return 0
    else
      echo "zero-downtime: $url and host $host_name is not yet available for requests, HTTP status code: $response"
    fi
  	sleep 1
    ((attempts = attempts + 1))
  done

  echo "zero-downtime: $url is not available for requests"
  exit 1
}

# Function to send requests to a given url
send_requests() {
  local thread="$1"
  local url="$2"
  local host_name="$3"
  local request_count=0
  echo "zero-downtime: thread ${thread}: Sending requests to $url to host $host_name"

  while true; do
    response=$(curl -sk -o /dev/null -w "%{http_code}" -H "Host: $host_name" "$url")
    ((request_count = request_count + 1))

    if [ "$response" != "200" ]; then
      sleep 5
      # If there is an error and the test-app Deployment, Gateway or Virtual Service still exists, the test is failed,
      # but if an error is received only when the one of those resources is deleted, the test is successful, because
      # without any of them the request will fail.
      if ! kubectl get deployment test-app -n default || ! kubectl get gateways test-gateway -n default || ! kubectl get virtualservices upgrade-test-vs -n default; then
        echo "zero-downtime: thread ${thread}: Test successful after $request_count requests. Stopping requests because on of the required resources is deleted."
        exit 0
      else
        echo "zero-downtime: thread ${thread}: Test failed after $request_count requests. Canceling requests because of HTTP status code $response"
        exit 1
      fi
    fi
  done
}

start() {
  # Start the requests in the background

  run_zero_downtime_requests "upgrade-test-vs" &
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
