<!-- cSpell: words argocd autocloud kustomize citest -->

# Generators

karmafun provides four custom generators beyond the kustomize builtins:

| Generator | Description |
| --------- | ----------- |
| [`GitConfigMapGenerator`](git-configmap.md) | ConfigMap populated with the current git repo URL and branch |
| [`KustomizationGenerator`](kustomization.md) | Resources produced by running a nested kustomization |
| [`SopsGenerator`](sops.md) | Kubernetes resources decrypted from SOPS-encrypted files |
| [`KCLGenerator`](kcl.md) | Resources generated from KCL code |

All builtin kustomize generators (`ConfigMapGenerator`, `SecretGenerator`, etc.)
are also available — see [Builtin plugins](../builtin.md).
