<!-- cSpell: words prune autocloud argocd -->

# Annotations reference

karmafun behavior is controlled through annotations on the **function
configuration** resource (the YAML file referenced by
`config.kubernetes.io/function`).

## `config.kubernetes.io/function`

Standard kustomize annotation that marks the resource as a KRM function and
specifies how to run it (`exec` or `container`):

```yaml
config.kubernetes.io/function: |
  exec:
    path: karmafun
```

---

## `config.karmafun.dev/cleanup`

Remove kustomize internal tracking annotations
(`internal.config.kubernetes.io/*`) from every resource touched by this
transformer. Use this when running transformers outside of a `kustomize build`
pipeline (i.e. with `kustomize fn run`) to prevent those annotations from being
written back to disk.

```yaml
config.karmafun.dev/cleanup: "true"
```

---

## `config.karmafun.dev/inject-local`

Bypass generation/transformation entirely and inject the **function config
itself** into the resource list, as if it had been generated. The
`config.karmafun.dev/inject-local` and `config.kubernetes.io/function`
annotations are stripped from the injected resource.

This is the _heredoc_ pattern: define arbitrary nested data directly in the
function file and use it as a replacement source:

```yaml
apiVersion: config.karmafun.dev/v1alpha1
kind: LocalConfiguration
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
```

---

## `config.karmafun.dev/local-config`

Mark a resource as _local configuration_ — part of the transformation pipeline
but not intended to be saved to disk. Combined with
`config.karmafun.dev/prune-local` on the **last** transformer, these resources
are removed before writing the output.

```yaml
config.karmafun.dev/local-config: "true"
```

Resources without this annotation (and without explicit
`config.karmafun.dev/path`) are saved to `.karmafun.yaml` in the configuration
directory. Add `.karmafun.yaml` to `.gitignore` to avoid accidental commits.

---

## `config.karmafun.dev/prune-local`

Remove all resources marked with `config.karmafun.dev/local-config` from the
output. Place this annotation on the **last** transformer in the pipeline:

```yaml
config.karmafun.dev/prune-local: "true"
```

---

## `config.karmafun.dev/path`

Override the filename used when writing generated resources to disk. Directories
in the path are created automatically.

```yaml
config.karmafun.dev/path: output/my-resources.yaml
```

Set to an empty string to write each resource to its own file following the
pattern `<namespace>/<Kind>_<name>.yaml`:

```yaml
config.karmafun.dev/path: ""
```

---

## `config.karmafun.dev/index`

Starting index used when writing multiple resources to a single file (relates to
`config.karmafun.dev/path`).

---

## `config.karmafun.dev/kind` and `config.karmafun.dev/apiVersion`

Override the `kind` and `apiVersion` of **generated** resources. Used primarily
by `SopsGenerator` when the function config is the encrypted resource itself:

```yaml
config.karmafun.dev/kind: "Secret"
config.karmafun.dev/apiVersion: "v1"
```
