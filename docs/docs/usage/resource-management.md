<!-- cSpell: words prune autocloud argocd karmafun -->

# Managing generated resources

Generated resources are written to disk by default. Use the following patterns
to control where they land (or whether they are saved at all).

## Keep resources (default)

Without any special annotation, generated resources are appended to
`.karmafun.yaml` in the configuration directory. Add `.karmafun.yaml` to
`.gitignore` to avoid accidental commits.

## Inject temporary resources and remove them

The most common pattern is to inject a _local_ resource for use in downstream
replacements, then remove it at the end of the pipeline:

1. Mark the generator with `config.karmafun.dev/local-config: "true"` so it is
   excluded from the final output.
2. Add `config.karmafun.dev/prune-local: "true"` to the **last** transformer to
   delete all local-config resources before writing output.

```yaml
# 01_configmap-generator.yaml
apiVersion: builtin
kind: ConfigMapGenerator
metadata:
  name: repo-values
  annotations:
    config.karmafun.dev/local-config: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
literals:
  - repoURL=https://github.com/myname/repo.git
  - targetRevision=main
---
# 02_replacement-transformer.yaml
apiVersion: builtin
kind: ReplacementTransformer
metadata:
  name: inject-values
  annotations:
    config.karmafun.dev/prune-local: "true"   # removes repo-values ConfigMap
    config.kubernetes.io/function: |
      exec:
        path: karmafun
replacements:
  - source:
      kind: ConfigMap
      name: repo-values
      fieldPath: data.repoURL
    targets:
      - select:
          kind: Application
        fieldPaths:
          - spec.source.repoURL
```

## Save resources to a specific file

Use `config.karmafun.dev/path` to control the output filename:

```yaml
config.karmafun.dev/path: infra/generated-resources.yaml
```

## One file per resource

Set `config.karmafun.dev/path` to an empty string to write each resource to its
own file using the pattern `<namespace>/<Kind>_<name>.yaml`:

```yaml
config.karmafun.dev/path: ""
```

For example, a DaemonSet named `kube-flannel-ds` in namespace `kube-flannel`
would be saved to `kube-flannel/daemonset_kube-flannel-ds.yaml`.

## Inject the function config itself as a resource

The `config.karmafun.dev/inject-local` annotation bypasses plugin execution and
injects the function config directly into the resource list. Combine it with
`config.karmafun.dev/local-config` and `config.karmafun.dev/prune-local` for a
self-contained, zero-file replacement source:

```yaml
apiVersion: config.karmafun.dev/v1alpha1
kind: PlatformValues
metadata:
  name: my-values
  annotations:
    config.karmafun.dev/inject-local: "true"
    config.karmafun.dev/local-config: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
data:
  domain: example.com
  revision: main
```

See [Annotations](annotations.md) for the full annotation reference.
