<!-- cSpell: words argocd sops krmfnsops lastmodified kustomize oyaml encrypted_regex -->

# SopsGenerator

`SopsGenerator` decrypts [SOPS](https://github.com/getsops/sops)-encrypted
files and injects the resulting Kubernetes resources into the pipeline. It is
an integration of [krmfnsops](https://github.com/kaweezle/krmfnsops).

Supported SOPS backends: age, PGP, AWS KMS, GCP KMS, Azure Key Vault.

## Mode 1 — external encrypted files

List one or more SOPS-encrypted YAML/JSON files in the `files` field:

```yaml
# argocd-secret-generator.yaml
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
metadata:
  name: argocd-secret-generator
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
files:
  - argocd-secret.yaml
  - another-secret.yaml
```

Each file must be a SOPS-encrypted YAML document containing valid Kubernetes
resources.

## Mode 2 — inline encrypted resource

Transform the encrypted resource file itself into a KRM generator by changing
its `apiVersion` and `kind`, and adding two annotations to restore the original
type on the generated output:

```yaml
# argocd-secret.yaml  (the encrypted file itself)
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
type: Opaque
metadata:
  name: argocd-secret
  annotations:
    config.karmafun.dev/kind: "Secret"
    config.karmafun.dev/apiVersion: "v1"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
stringData:
  admin.password: ENC[AES256_GCM,data:...,type:str]
  webhook.github.secret: ENC[AES256_GCM,data:...,type:str]
sops:
  age:
    - recipient: age166k86d56...
      enc: |
        -----BEGIN AGE ENCRYPTED FILE-----
        ...
        -----END AGE ENCRYPTED FILE-----
  lastmodified: "2023-02-06T11:36:44Z"
  mac: ENC[AES256_GCM,data:...,type:str]
  encrypted_regex: ^(data|stringData|.*_keys?|admin|adminKey|password)$
  version: 3.7.3
```

Then reference it directly in `kustomization.yaml`:

```yaml
generators:
  - argocd-secret.yaml
```

!!! warning
    Inline mode disables SOPS Message Authentication Code (MAC) verification.
    Use it at your own risk.

## Generated resource annotations

The generator sets `config.karmafun.dev/inject-local: "true"` on decrypted
resources so they can be pruned at the end of the pipeline with
`config.karmafun.dev/prune-local: "true"` if needed.
