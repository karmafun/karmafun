# Installation

## Automatic (Linux / macOS)

```console
curl -sLS https://raw.githubusercontent.com/karmafun/karmafun/main/get.sh | /bin/sh
```

## Manual binary download

Replace `<VERSION>` with the desired release tag (e.g. `v0.4.3`):

```console
KARMAFUN_VERSION="v0.4.3"
curl -sLo /usr/local/bin/karmafun \
  https://github.com/karmafun/karmafun/releases/download/${KARMAFUN_VERSION}/karmafun_${KARMAFUN_VERSION}_linux_amd64
chmod +x /usr/local/bin/karmafun
```

Binaries for macOS (`darwin_amd64`, `darwin_arm64`) and Windows are also
available on the [Releases](https://github.com/karmafun/karmafun/releases) page,
along with Alpine packages (`.apk`).

## Container image

```yaml
config.kubernetes.io/function: |
  container:
    image: ghcr.io/karmafun/karmafun:v0.4.3
```

Use the container image instead of the `exec` path when running in environments
where you cannot install a binary.

## Kustomize plugin registration

`karmafun setup` (or `karmafun install`) creates the required symlinks under
the kustomize plugin directory so that kustomize can discover all plugin kinds:

```console
karmafun setup
```

This is only needed when using karmafun as a _legacy_ exec plugin (i.e., without
the `config.kubernetes.io/function` annotation).
