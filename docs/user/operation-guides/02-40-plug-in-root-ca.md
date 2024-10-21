---
title: Plug in non-default Root CA into Istiod
---

In case you want to ensure that Istio pilot trusts certificates of a server to which it establishes connections (e.g. for fetching of JWKS), and the server certificate is signed by a generally not trusted Certificate Authority (CA) you may want to plug in the CA certificate directly into the istiod Deployment.

This can be done by creating a ConfigMap that contains the PEM of the CA certificate and mounting it as volume in the istiod deployment. For convenience, the following script uses kubectl to achieve this:

```sh
#!/usr/bin/env bash

set -eaux pipefail

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: istio-jwks-extra-cacerts
  namespace: istio-system
data:
  extra.pem: |
    -----BEGIN CERTIFICATE-----
    {CERTIFICATE_PEM_DATA}
    -----END CERTIFICATE-----
EOF

kubectl patch deployment -n istio-system istiod --type json -p '[{"op": "add", "path": "/spec/template/spec/volumes/-", "value": {"name": "extracacerts", "configMap": {"defaultMode": 420, "optional": true, "name": "istio-jwks-extra-cacerts"}}}]'
kubectl patch deployment -n istio-system istiod --type json -p '[{"op": "add", "path": "/spec/template/spec/containers/0/volumeMounts/-", "value": {"mountPath": "/cacerts", "name": "extracacerts", "readOnly": true}}]'
```

The script has 3 steps:

1. Create a ConfigMap `istio-jwks-extra-cacerts` in the `istio-system` namespace containing the Certificate PEM.
2. Patch the `istiod` deployment to include the ConfigMap as a mounted volume `extracacerts`.
3. Patch the `istiod` deployment mounting the volume under `/cacerts`.

After a rollout restart, `istiod` should recognize the server certificate as trusted.

