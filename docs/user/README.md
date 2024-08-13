# Istio Module
Learn more about the Istio module. Use it to manage and configure the Istio service mesh.

## What Is Istio?

[Istio](https://istio.io/latest/) is an open-source service mesh that provides a uniform way to manage, connect, and secure microservices. It helps to manage traffic, enhance security capabilities, and provide telemetry data for understanding service behavior.

The latest release includes the following versions of Istio and Envoy:  

**Istio version:** 1.23.2

**Envoy version:** 1.31.2

## Features

## Scope

## Architecture

### Istio Operator

Within the Istio module, Istio Operator handles the management and configuration of the Istio service mesh. It contains one controller, referred to as Istio Controller.

### Istio Controller

Istio Controller manages the installation of Istio as defined by the Istio custom resource (CR). It is responsible for tasks such as installing, upgrading, and uninstalling Istio, as well as restarting workloads with a proxy sidecar to ensure they are using the correct version of Istio.

## API / Custom Resource Definitions

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation.

## Resource Consumption



--------------
