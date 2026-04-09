<!-- cSpell: words myorg myrepo -->

# GitConfigMapGenerator

`GitConfigMapGenerator` works identically to kustomize's
[`ConfigMapGenerator`](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_configmapgenerator_)
but automatically populates two additional entries from the local git
repository:

| Key              | Value                                                           |
| ---------------- | --------------------------------------------------------------- |
| `repoURL`        | URL of the remote specified by `remoteName` (default: `origin`) |
| `targetRevision` | Short name of the current branch (`HEAD`)                       |

This generator is particularly useful for Argo CD application customization
where you need to propagate the current fork's URL and branch into every
application.

## Configuration

```yaml
apiVersion: builtin
kind: GitConfigMapGenerator
metadata:
  name: repo-values
  annotations:
    config.karmafun.dev/local-config: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
# Optional: defaults to "origin"
remoteName: origin
# Any standard ConfigMapGenerator field is supported
# literals:
#   - extraKey=extraValue
```

All fields from kustomize's `ConfigMapGenerator` (`literals`, `files`, `envs`,
`options`, …) are accepted in addition to `remoteName`.

## Output

The generated ConfigMap contains at minimum:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: repo-values
data:
  repoURL: git@github.com:myorg/myrepo.git
  targetRevision: feature/my-branch
```

## Full pipeline example

Inject the current repo's values into all Argo CD applications:

```yaml
# functions/01_git-values.yaml
apiVersion: builtin
kind: GitConfigMapGenerator
metadata:
  name: git-values
  annotations:
    config.karmafun.dev/local-config: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
---
# functions/02_replace-sources.yaml
apiVersion: builtin
kind: ReplacementTransformer
metadata:
  name: replace-sources
  annotations:
    config.karmafun.dev/cleanup: "true"
    config.karmafun.dev/prune-local: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
replacements:
  - source:
      kind: ConfigMap
      name: git-values
      fieldPath: data.repoURL
    targets:
      - select:
          kind: Application
          annotationSelector: "autocloud/local=true"
        fieldPaths:
          - spec.source.repoURL
  - source:
      kind: ConfigMap
      name: git-values
      fieldPath: data.targetRevision
    targets:
      - select:
          kind: Application
          annotationSelector: "autocloud/local=true"
        fieldPaths:
          - spec.source.targetRevision
```

```console
kustomize fn run --enable-exec --fn-path functions applications
```
