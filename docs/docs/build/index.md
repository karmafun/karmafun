<!-- cSpell: words argocd kustomize sops gotmpl appstage sprig gojq -->

# karmafun build

`karmafun build` is a drop-in replacement for `kustomize build` that adds:

- **Values and secrets injection** — a `values.yaml` and an optional
  SOPS-encrypted `secrets.sops.yaml` are deep-merged and exposed as Go template
  variables.
- **Go template rendering** — any file ending with `.tmpl` or `.gotmpl` in the
  kustomization directory is rendered as a [Go text/template][gotpl] (with all
  [Sprig][sprig] functions) before kustomize processes it.

All karmafun plugins (and any standard kustomize plugin) work as usual inside
the build.

## Usage

```console
karmafun build [flags] <kustomization directory>
```

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--values-file` | `values.yaml` | Plain-text platform values file |
| `--secrets-file` | `secrets.sops.yaml` | SOPS-encrypted secrets file |
| `--output-directory` / `-o` | _(stdout)_ | Write each resource to `Kind-name.yaml` in this directory |
| `--log-level` | `info` | `debug`, `info`, `warn`, `error` |
| `--log-json` | `false` | Emit logs in JSON format |

## Values and secrets files

Both files use the `PlatformValues` resource format:

```yaml
# values.yaml
apiVersion: config.karmafun.dev/v1alpha1
kind: PlatformValues
metadata:
  name: my-values
data:
  domain_suffix: example.com
  project:
    name: my-project
  argocd:
    target_revision: main
```

The secrets file has the same structure but must be SOPS-encrypted:

```console
sops -e secrets.dec.sops.yaml > secrets.sops.yaml
```

Secrets values take precedence over plain values on deep-merge. Absent files are
silently ignored.

## Go templates

Any resource file ending with `.tmpl` or `.gotmpl` is rendered as a Go template.
The merged values are available under `.Values`:

| Expression | Description |
| ---------- | ----------- |
| `{{ .Values.data.key }}` | Top-level key |
| `{{ .Values.data.nested.key }}` | Nested key |

All [Sprig][sprig] helper functions (`upper`, `trim`, `toJson`, …) are available.

**Example** — `application.yaml.gotmpl`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  source:
    repoURL: git@github.com:{{ .Values.data.project.github.organization }}/{{ .Values.data.project.github.repo }}.git
    targetRevision: {{ .Values.data.argocd.target_revision }}
    path: deploy/k8s/{{ .Values.data.argocd.base_path }}
```

## Sample kustomization

The [`samples/kustomization`][sample] directory demonstrates the full workflow:

```
samples/kustomization/
├── appstage-00-bootstrap/
│   ├── application.yaml.gotmpl   # Uses .Values.data.*
│   └── kustomization.yaml
├── values.yaml
├── secrets.dec.sops.yaml         # Unencrypted reference
├── secrets.sops.yaml             # SOPS-encrypted
└── test_kustomization.sh
```

To run the sample:

```console
# Extract the sample age key
gojq -r --yaml-input '.data.sops["age_key.txt"]' \
  ./samples/kustomization/secrets.dec.sops.yaml \
  >> ~/.config/sops/age/keys.txt

karmafun build samples/kustomization/appstage-00-bootstrap
```

[gotpl]: https://pkg.go.dev/text/template
[sprig]: https://masterminds.github.io/sprig/
[sample]: https://github.com/karmafun/karmafun/tree/main/samples/kustomization
