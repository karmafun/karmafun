<!-- cSpell: words kustomize argocd uninode citest -->

# KustomizationGenerator

`KustomizationGenerator` is the kustomize equivalent of
`HelmChartInflationGenerator`. It runs `kustomize build` on an arbitrary
directory or URL and injects the resulting resources into the current pipeline.

## Configuration

```yaml
apiVersion: builtin
kind: KustomizationGenerator
metadata:
  name: my-infra
  annotations:
    config.karmafun.dev/path: "infra.yaml"   # save output here
    config.kubernetes.io/function: |
      exec:
        path: karmafun
kustomizeDirectory: https://github.com/myorg/myrepo.git//packages/infra?ref=main
```

| Field                | Description |
| -------------------- | ----------- |
| `kustomizeDirectory` | Path or URL passed to `kustomize build`. Supports any URL format kustomize accepts, including Git remote URLs with sub-paths and refs. |

## Path resolution

`karmafun` is invoked from the directory where `kustomize fn run` is executed,
**not** from the directory containing the function file. Use absolute paths or
remote URLs to avoid ambiguity.

## Output control

Combine with `config.karmafun.dev/path` to control where resources are written:

```yaml
# Save all resources to a single file
config.karmafun.dev/path: "uninode.yaml"

# Save each resource to its own file (<namespace>/<Kind>_<name>.yaml)
config.karmafun.dev/path: ""
```

## Example: inflate a remote kustomization

```yaml
apiVersion: builtin
kind: KustomizationGenerator
metadata:
  name: uninode
  annotations:
    config.karmafun.dev/path: "uninode.yaml"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
kustomizeDirectory: https://github.com/antoinemartin/autocloud.git//packages/uninode?ref=deploy/citest
```

Running:

```console
kustomize fn run --enable-exec --fn-path functions applications
```

produces `applications/uninode.yaml` containing all resources from the remote
kustomization.
