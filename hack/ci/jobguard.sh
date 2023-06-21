#!/bin/bash

set -e

JOB_NAME_PATTERN=${JOB_NAME_PATTERN:-"(pull-.*-build)"}
TIMEOUT=${JOBGUARD_TIMEOUT:-"15m"}

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
  "-expected-contexts-regexp=$JOB_NAME_PATTERN"
)

if [ -x "/prow-tools/jobguard" ]; then
  jobguard "${args[@]}"
else
  echo "Can not find jobguard in PATH"
  exit 1
fi
