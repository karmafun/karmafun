<!-- cSpell: words distros pacman -->

# Installation

## Automatic (Linux / macOS)

```console
curl -sLS https://raw.githubusercontent.com/karmafun/karmafun/main/get.sh | /bin/sh
```

## Manual binary download

Replace `<VERSION>` with the desired release tag (e.g. `{git_latest_release}`)
and run the following commands to download the binary:

```console
KARMAFUN_VERSION="{git_latest_release}"
curl -sLo /usr/local/bin/karmafun \
  https://github.com/karmafun/karmafun/releases/download/${KARMAFUN_VERSION}/karmafun_${KARMAFUN_VERSION}_linux_amd64
chmod +x /usr/local/bin/karmafun
```

Binaries for macOS (`darwin_amd64`, `darwin_arm64`) and Windows are also
available on the [Releases](https://github.com/karmafun/karmafun/releases) page,
along with native packages for Alpine, Debian/Ubuntu, RPM-based distros, and
Arch Linux.

## Package managers

=== "Alpine"

    Add the signing key and repository, then install:

    ```console
    cd /etc/apk/keys && curl -sLO "https://github.com/karmafun/karmafun/releases/latest/download/kaweezle-devel@kaweezle.com-c9d89864.rsa.pub"
    echo "https://static.iknite.app/release/" >> /etc/apk/repositories
    apk update
    apk add karmafun
    ```

=== "Debian / Ubuntu"

    Download and install the `.deb` package from the
    [Releases](https://github.com/karmafun/karmafun/releases) page:

    ```console
    KARMAFUN_VERSION="{git_latest_release}"
    ARCH=$(dpkg --print-architecture)   # amd64 or arm64
    curl -sLo karmafun.deb \
      "https://github.com/karmafun/karmafun/releases/download/${KARMAFUN_VERSION}/karmafun_${KARMAFUN_VERSION}_linux_${ARCH}.deb"
    sudo dpkg -i karmafun.deb
    rm karmafun.deb
    ```

=== "RPM-based (Fedora / RHEL / openSUSE)"

    Download and install the `.rpm` package:

    ```console
    KARMAFUN_VERSION="{git_latest_release}"
    ARCH=$(uname -m)                    # x86_64 or aarch64
    sudo rpm -ivh \
      "https://github.com/karmafun/karmafun/releases/download/${KARMAFUN_VERSION}/karmafun_${KARMAFUN_VERSION}_linux_${ARCH}.rpm"
    ```

=== "Arch Linux"

    Download and install the `.pkg.tar.zst` package:

    ```console
    KARMAFUN_VERSION="{git_latest_release}"
    ARCH=$(uname -m)                    # x86_64 or aarch64
    curl -sLo karmafun.pkg.tar.zst \
      "https://github.com/karmafun/karmafun/releases/download/${KARMAFUN_VERSION}/karmafun_${KARMAFUN_VERSION}_linux_${ARCH}.pkg.tar.zst"
    sudo pacman -U karmafun.pkg.tar.zst
    rm karmafun.pkg.tar.zst
    ```

## Container image

```yaml
config.kubernetes.io/function: |
  container:
    image: ghcr.io/karmafun/karmafun:{git_latest_release}
```

Use the container image instead of the `exec` path when running in environments
where you cannot install a binary.

## Kustomize plugin registration

`karmafun setup` (or `karmafun install`) creates the required symlinks under the
kustomize plugin directory so that kustomize can discover all plugin kinds:

```console
karmafun setup
```

This is only needed when using karmafun as a _legacy_ exec plugin (i.e., without
the `config.kubernetes.io/function` annotation).
