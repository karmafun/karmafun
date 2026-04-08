<!-- cSpell: words argocd kustomize -->

# Plugins

karmafun exposes two categories of plugins as KRM functions:

- **[Builtin plugins](builtin.md)** — the standard kustomize transformers and
  generators, made available to `kustomize fn run`.
- **Additional generators** — [`GitConfigMapGenerator`](generators/git-configmap.md),
  [`KustomizationGenerator`](generators/kustomization.md),
  [`SopsGenerator`](generators/sops.md), [`KCLGenerator`](generators/kcl.md).
- **Additional transformers** —
  [`ReplacementTransformer`](transformers/extended-replacement.md) (extended),
  [`RemoveTransformer`](transformers/remove.md),
  [`KCLTransformer`](transformers/kcl.md).

## Referencing a plugin

Every plugin is selected by the `kind` field of the function configuration:

```yaml
apiVersion: builtin
kind: LabelTransformer          # ← selects the plugin
metadata:
  name: add-labels
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
labels:
  app.kubernetes.io/managed-by: karmafun
fieldSpecs:
  - path: metadata/labels
    create: true
```

Additional generators and transformers use their own `apiVersion`:

| apiVersion                   | Plugins                                    |
| ---------------------------- | ------------------------------------------ |
| `builtin`                    | All kustomize builtin plugins              |
| `builtin` or `karmafun.dev/v1alpha1` | `RemoveTransformer`               |
| `karmafun.dev/v1alpha1`      | `SopsGenerator`                            |
| `kcl.dev/v1alpha1`           | `KCLRun`, `KCLTransformer`                 |
