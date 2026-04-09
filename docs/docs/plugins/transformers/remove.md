<!-- cSpell: words argocd kustomize -->

# RemoveTransformer

`RemoveTransformer` removes resources that match one or more selectors. It is
the logical inverse of most transformers — instead of modifying resources, it
deletes them from the pipeline.

## Configuration

```yaml
apiVersion: builtin
kind: RemoveTransformer
metadata:
  name: remove-local-config
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
targets:
  - annotationSelector: config.karmafun.dev/local-config
```

Each entry in `targets` follows the
[kustomize patch target convention](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#field-name-patches)
and supports:

| Field               | Description                              |
| ------------------- | ---------------------------------------- |
| `group`             | API group                                |
| `version`           | API version                              |
| `kind`              | Resource kind                            |
| `name`              | Resource name                            |
| `namespace`         | Resource namespace                       |
| `labelSelector`     | Label selector string                    |
| `annotationSelector`| Annotation selector string               |

At least one target must be specified.

## Alternative

The same result can be achieved with kustomize's built-in
`PatchStrategicMergeTransformer` using a `$patch: delete` field. `RemoveTransformer`
is more explicit and easier to read.

## Example: clean up after a multi-step pipeline

When `karmafun` is not the last function in a multi-tool pipeline, you cannot
rely on `config.karmafun.dev/prune-local`. Use `RemoveTransformer` instead:

```yaml
apiVersion: builtin
kind: RemoveTransformer
metadata:
  name: cleanup
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: ../../karmafun
targets:
  - annotationSelector: config.karmafun.dev/local-config
  - kind: ConfigMap
    name: temporary-values
```
