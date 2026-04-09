<!-- cSpell: words argocd kustomize argo-repo-server -->

# Argo CD integration

!!! note
    karmafun is primarily a _pre-commit_ tool. It is designed to make structural
    changes to configuration **before** it is committed and handed to GitOps. Use
    it inside Argo CD only when that workflow is not possible.

## Prerequisites

To use karmafun inside Argo CD:

1. Make the `karmafun` binary available inside the `argo-repo-server` pod.
2. Configure Argo CD to pass `--enable-alpha-plugins --enable-exec` to kustomize.

## Adding the binary

The [Argo CD documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/custom_tools/)
describes several ways to install custom tools. The most robust is to build a
custom image:

```dockerfile
FROM argoproj/argocd:latest

ARG KARMAFUN_VERSION=v0.4.3

USER root

RUN apt-get update && \
    apt-get install -y curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    curl -sLo /usr/local/bin/karmafun \
      https://github.com/karmafun/karmafun/releases/download/${KARMAFUN_VERSION}/karmafun_${KARMAFUN_VERSION}_linux_amd64 && \
    chmod +x /usr/local/bin/karmafun

USER argocd
```

## Enabling exec plugins in Argo CD

Patch the `argocd-cm` ConfigMap:

```yaml
# argocd-cm.yaml  (strategic merge patch)
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
data:
  kustomize.buildOptions: "--enable-alpha-plugins --enable-exec"
```

## Function config

Use the `exec` form to reference karmafun inside function configurations:

```yaml
config.kubernetes.io/function: |
  exec:
    path: karmafun
```

Or use the container image to avoid binary installation:

```yaml
config.kubernetes.io/function: |
  container:
    image: ghcr.io/karmafun/karmafun:v0.4.3
```
