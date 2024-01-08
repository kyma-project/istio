# Tests

## Overview

This folder contains Istio integration tests for Kyma dashboard.

## Prerequisites

If you want to use an existing cluster, you must copy your cluster's kubeconfig file to `fixtures/kubeconfig.yaml`.

## Installation

To install the dependencies, run the `npm clean install` command.

## Test Development Using Headless Mode with Chrome Browser

### Using a Local Kyma Dashboard Instance

Start a k3d cluster:

```bash
npm run start-k3d
```

Start the local Kyma Dashboard instance:

```bash
npm run start-dashboard
```

#### Run Tests

```bash
npm run test
```

#### Run Cypress UI Tests in the Test Runner Mode

```bash
npm run start
```

### Using a Remote Kyma Dashboard Instance

#### Optional: Log In to a Cluster Using OIDC

If a cluster requires OIDC authentication, include the additional arguments **CYPRESS_OIDC_PASS** and **CYPRESS_OIDC_USER** while running the npm scripts.

#### Run Tests
```bash
CYPRESS_OIDC_PASS={YOUR_PASSWORD} CYPRESS_OIDC_USER={YOUR_USERNAME} CYPRESS_DOMAIN={REMOTE_CLUSTER_DASHBOARD_DOMAIN} npm run test
```

#### Run Cypress UI Tests in the Test Runner Mode

```bash
CYPRESS_OIDC_PASS={YOUR_PASSWORD} CYPRESS_OIDC_USER={YOUR_USERNAME} CYPRESS_DOMAIN={REMOTE_CLUSTER_DASHBOARD_DOMAIN} npm run start
```

## Run Tests in Continuous Integration System

Start a k3d cluster and run the tests:

```bash
./scripts/k3d-ci-kyma-dashboard-integration.sh
```
