#!/bin/bash

set -e
WAIT_FOR_JOB_NAME

WAIT_FOR_JOB_NAME=${WAIT_FOR_JOB_NAME:-"(post-.*-build)|(pull-.*-build)"}
TIMEOUT=${JOBGUARD_TIMEOUT:-"10m"}

if [[ "${BUILD_TYPE}" == "pr" ]]; then
    BASE_REF=${PULL_PULL_SHA}
else
    BASE_REF=${PULL_BASE_SHA}
fi

args=(
  "-github-endpoint=http://ghproxy"
  "-github-endpoint=https://api.github.com"
  "-github-token-path=/etc/github/token"
  "-fail-on-no-contexts=false"
  "-timeout=$TIMEOUT"
  "-org=$REPO_OWNER"
  "-repo=$REPO_NAME"
  "-base-ref=$BASE_REF"
  "-expected-contexts-regexp=$WAIT_FOR_JOB_NAME"
)

jobguard "${args[@]}"