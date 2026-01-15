# Configurable Istio Images

This document describes how Istio component images are configured using environment variables, enabling support for different image variants like FIPS-compliant or OS-specific builds.

## Overview

Kyma Istio Operator uses environment variables to configure full image references for each Istio component. This approach provides flexibility to use different image registries, image names, and tags for each component independently.

The following Istio component images are configurable:

| Component       | Environment Variable | IstioOperator Path          |
|-----------------|----------------------|-----------------------------|
| Pilot           | `PILOT_IMAGE`        | `values.pilot.image`        |
| Proxy (sidecar) | `PROXY_V2_IMAGE`     | `values.global.proxy.image` |
| CNI             | `INSTALL_CNI_IMAGE`  | `values.cni.image`          |

## Image Format

Each environment variable must contain a full image reference in the format:

```
<registry>/<repository>/<image-name>:<tag>
```

**Example:**
```
europe-docker.pkg.dev/kyma-project/prod/external/istio/pilot:1.24.0-distroless
```