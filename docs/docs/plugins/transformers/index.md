<!-- cSpell: words argocd kustomize -->

# Transformers

karmafun provides three custom transformers beyond the kustomize builtins:

| Transformer | Description |
| ----------- | ----------- |
| [`ReplacementTransformer`](extended-replacement.md) | Extended replacement with structured-content support (YAML / JSON / TOML / INI / regex / encoding) |
| [`RemoveTransformer`](remove.md) | Remove resources matching a selector |
| [`KCLTransformer`](kcl.md) | Transform resources using KCL code |

All builtin kustomize transformers (`LabelTransformer`, `PatchTransformer`, etc.)
are also available — see [Builtin plugins](../builtin.md).
