<!-- cSpell: words kustomize argocd kpt sops yq -->

# Running karmafun

karmafun is invoked as a KRM function. There are two ways to reference it in a
function configuration: as a local **exec** binary or as a **container image**.

## Referencing karmafun

### exec (local binary)

```yaml
config.kubernetes.io/function: |
  exec:
    path: karmafun
```

The binary must be in `$PATH` or you can use a relative/absolute path. In the
tests directory, function files reference the binary two levels up:

```yaml
config.kubernetes.io/function: |
  exec:
    path: ../../karmafun
```

### container image

```yaml
config.kubernetes.io/function: |
  container:
    image: ghcr.io/karmafun/karmafun:{git_latest_release}
```

No local installation is required. The `--enable-exec` flag is not needed when
using the container form.

---

## With `kustomize fn run`

[`kustomize fn run`](https://kubectl.docs.kubernetes.io/references/kustomize/kustomize_fn_run/)
applies all function configs found in a given directory against a set of
resource files **in-place** (files are modified on disk).

**Directory layout:**

```
my-project/
├── applications/     # Kubernetes resource files to transform
│   └── app.yaml
└── functions/        # Function config files (applied in lexical order)
    ├── 01_configmap-generator.yaml
    └── 02_replacement-transformer.yaml
```

**Run:**

```console
kustomize fn run --enable-exec --fn-path functions applications
```

`--enable-exec` is required when using the `exec` form. It is not needed for the
container form.

### Example: patch transformer

Given a function file `functions/fn-patch.yaml`:

```yaml
apiVersion: builtin
kind: PatchTransformer
metadata:
  name: update-repo
  annotations:
    config.karmafun.dev/cleanup: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
patch: |-
  - op: replace
    path: /spec/source/repoURL
    value: https://github.com/myname/repo.git
  - op: replace
    path: /spec/source/targetRevision
    value: feature/experiment
target:
  group: argoproj.io
  version: v1alpha1
  kind: Application
  annotationSelector: "autocloud/local=true"
```

Running the command above modifies the matching application files in
`applications/` in place.

### Example: ConfigMap generator + replacement transformer

Two numbered function files are applied in order:

```yaml
# functions/01_configmap-generator.yaml
apiVersion: builtin
kind: ConfigMapGenerator
metadata:
  name: configuration-map
  namespace: argocd
  annotations:
    config.karmafun.dev/local-config: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
literals:
  - repoURL=https://github.com/myname/repo.git
  - targetRevision=main
```

```yaml
# functions/02_replacement-transformer.yaml
apiVersion: builtin
kind: ReplacementTransformer
metadata:
  name: replacement-transformer
  namespace: argocd
  annotations:
    config.karmafun.dev/prune-local: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
replacements:
  - source:
      kind: ConfigMap
      fieldPath: data.repoURL
    targets:
      - select:
          kind: Application
          annotationSelector: "autocloud/local=true"
        fieldPaths:
          - spec.source.repoURL
          - spec.source.helm.parameters.[name=common.repoURL].value
  - source:
      kind: ConfigMap
      fieldPath: data.targetRevision
    targets:
      - select:
          kind: Application
          annotationSelector: "autocloud/local=true"
        fieldPaths:
          - spec.source.targetRevision
          - spec.source.helm.parameters.[name=common.targetRevision].value
```

!!! tip Prefix function file names with numbers (`01_`, `02_`, …) to control
execution order. Alternatively, separate multiple functions in a single file
with `---` (not compatible with `kpt fn eval`).

---

## With `kpt fn eval`

[kpt](https://kpt.dev/) applies functions one at a time using a pipeline model.
Functions read from stdin and write to stdout, which `kpt fn source` /
`kpt fn sink` handle for reading and writing resource files.

**Run a single function:**

```console
kpt fn eval applications --exec karmafun --fn-config functions/fn-patch.yaml
```

**Run a pipeline of functions** (from `tests/test_karmafun_kpt.sh`):

```bash
temp=$(mktemp)
# Collect resources into a ResourceList
kpt fn source original > "$temp"

# Apply each function config sequentially
for f in functions/*.yaml; do
    cat "$temp" | kpt fn eval - --exec karmafun --fn-config "$f" > "$temp.new"
    mv "$temp.new" "$temp"
done

# Write results back to disk
cat "$temp" | kpt fn sink applications
```

Each function config is applied independently in order; the output of one
becomes the input of the next.

!!! note When using kpt, do **not** combine multiple functions in a single file
with `---`. Each function config must be in its own file.

---

## Testing your setup

The `tests/` directory contains self-contained test cases, each with:

- `original/` — input resources
- `functions/` — function configs to apply
- `expected/` — expected output (compared with `yq`)

To run all tests with kustomize:

```console
go build -o karmafun
./tests/test_karmafun.sh
```

To run all tests with kpt:

```console
go build -o karmafun
./tests/test_karmafun_kpt.sh
```

Both scripts require `kustomize` (or `kpt`), `yq`, and the `karmafun` binary in
the repository root.
