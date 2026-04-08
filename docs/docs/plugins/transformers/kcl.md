<!-- cSpell: words argocd kustomize kcl kusion krm cert -->

# KCLTransformer

`KCLTransformer` transforms existing Kubernetes resources using
[KCL (Kusion Configuration Language)](https://kcl-lang.io/) code. It can read
existing resources from the input, modify them, and optionally generate new ones.

For more information on the configuration options see the
[krm-kcl documentation](https://github.com/kcl-lang/krm-kcl).

## Configuration

```yaml
apiVersion: kcl.dev/v1alpha1
kind: KCLTransformer
metadata:
  name: my-transformer
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
spec:
  source: ./kcl/transform.k
```

| Field          | Description |
| -------------- | ----------- |
| `spec.source`  | Inline KCL code or path to a `.k` file |
| `spec.dependencies` | KCL module dependencies (OCI or registry format) |

## Example: generate cert-manager resources

```yaml
apiVersion: kcl.dev/v1alpha1
kind: KCLTransformer
metadata:
  name: generate-certificates
  annotations:
    config.karmafun.dev/prune-local: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
spec:
  dependencies: |
    cert-manager = "0.3.0"
  source: ./functions/transformer.k
```

The transformer KCL code receives existing resources as input and can return
modified or new resources.
