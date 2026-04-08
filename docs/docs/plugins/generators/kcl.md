<!-- cSpell: words argocd kustomize kcl kusion krm -->

# KCLGenerator

`KCLGenerator` (`KCLRun`) generates Kubernetes resources from
[KCL (Kusion Configuration Language)](https://kcl-lang.io/) code. KCL is a
constraint-based record and functional language designed for configuration and
policy scenarios.

For more information on the configuration options see the
[krm-kcl documentation](https://github.com/kcl-lang/krm-kcl).

## Inline KCL

```yaml
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: example
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "example"
      data.key = "value"
    }
```

## KCL from file

```yaml
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: example
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
spec:
  source: ./kcl/configmap.k
```

The path is relative to the directory where `kustomize fn run` is executed.

## Dependencies

Use the `dependencies` field to pull in KCL modules:

```yaml
spec:
  dependencies: |
    cert-manager = "0.3.0"
  source: ./kcl/generate.k
```
