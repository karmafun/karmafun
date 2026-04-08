<!-- cSpell: words kustomize argocd -->

# Builtin kustomize plugins

All standard kustomize transformers and generators are available as KRM
functions via karmafun. Use `apiVersion: builtin` and set `kind` to the plugin
name. The function config fields are identical to the kustomize built-in plugin
fields.

## Transformers

| Kind                           | Description                                         | Docs |
| ------------------------------ | --------------------------------------------------- | ---- |
| `AnnotationsTransformer`       | Add / update annotations on resources               | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_annotationstransformer_) |
| `HashTransformer`              | Append a content hash to resource names             | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_hashtransformer_) |
| `ImageTagTransformer`          | Update container image tags / digests               | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_imagetagtransformer_) |
| `LabelTransformer`             | Add / update labels on resources                    | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_labeltransformer_) |
| `NamespaceTransformer`         | Set the namespace on all resources                  | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_namespacetransformer_) |
| `PatchJson6902Transformer`     | Apply RFC 6902 JSON patches                         | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_patchjson6902transformer_) |
| `PatchStrategicMergeTransformer` | Apply strategic merge patches                     | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_patchstrategicmergetransformer_) |
| `PatchTransformer`             | Apply JSON 6902 or strategic merge patches          | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_patchtransformer_) |
| `PrefixTransformer`            | Add a prefix to resource names                      | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_prefixtransformer_) |
| `SuffixTransformer`            | Add a suffix to resource names                      | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_suffixtransformer_) |
| `PrefixSuffixTransformer`      | Add prefix and suffix in one pass                   | — |
| `ReplicaCountTransformer`      | Update replica counts                               | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_replicacounttransformer_) |
| `ValueAddTransformer`          | Add a value to a field                              | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_valueaddtransformer_) |
| `ReplacementTransformer`       | Replace field values using sources (**extended**)   | [↗](transformers/extended-replacement.md) |
| `RemoveTransformer`            | Remove resources by selector (**custom**)           | [↗](transformers/remove.md) |
| `KCLTransformer`               | Transform resources with KCL code (**custom**)      | [↗](transformers/kcl.md) |

## Generators

| Kind                          | Description                                         | Docs |
| ----------------------------- | --------------------------------------------------- | ---- |
| `ConfigMapGenerator`          | Generate ConfigMaps from literals / files / envs    | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_configmapgenerator_) |
| `SecretGenerator`             | Generate Secrets from literals / files / envs       | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_secretgenerator_) |
| `HelmChartInflationGenerator` | Render a Helm chart into plain resources            | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_helmchartinflationgenerator_) |
| `IAMPolicyGenerator`          | Generate IAM policy documents                       | [↗](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/) |
| `GitConfigMapGenerator`       | ConfigMap populated with current git repo values (**custom**) | [↗](generators/git-configmap.md) |
| `KustomizationGenerator`      | Generate resources from a kustomization (**custom**)| [↗](generators/kustomization.md) |
| `SopsGenerator`               | Decrypt SOPS-encrypted files into resources (**custom**) | [↗](generators/sops.md) |
| `KCLGenerator`                | Generate resources from KCL code (**custom**)       | [↗](generators/kcl.md) |

## Example: adding labels in-place

```yaml
apiVersion: builtin
kind: LabelTransformer
metadata:
  name: add-labels
  annotations:
    config.karmafun.dev/cleanup: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
labels:
  app.kubernetes.io/managed-by: karmafun
fieldSpecs:
  - path: metadata/labels
    create: true
```

```console
kustomize fn run --enable-exec --fn-path functions applications
```
