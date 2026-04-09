<!-- cSpell: words autocloud citest argocd krmfnsops gotmpl appstage uninode websecure instana holepunch sshconfig kusion -->

# karmafun

**karmafun** is a
[kustomize plugin](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/)
and a `kustomize build` wrapper providing a set of [KRM Functions] that perform
in-place transformations on Kubernetes resource files.

## Why karmafun?

`kustomize fn run` enables _in-place_ transformation of KRM resources — perfect
for making structural changes to a GitOps repository without adding extra
nesting layers. Unfortunately, kustomize's built-in transformers and generators
are not available to `kustomize fn run`, which requires an external `container`
or `exec` binary.

karmafun fills that gap by providing:

- All kustomize [builtin transformers and generators](plugins/builtin.md) as KRM
  functions.
- Additional [generators](plugins/generators/index.md): `GitConfigMapGenerator`,
  `KustomizationGenerator`, `SopsGenerator`, `KCLGenerator`.
- Additional [transformers](plugins/transformers/index.md): `RemoveTransformer`,
  `KCLTransformer`, and an extended `ReplacementTransformer` with
  structured-content support.
- A [`karmafun build`](usage/karmafun-build.md) command extending
  `kustomize build` with Go template rendering and SOPS-encrypted secrets.

## Quick start

### 1. Install

```console
curl -sLS https://raw.githubusercontent.com/karmafun/karmafun/main/get.sh | /bin/sh
```

See [Installation](installation.md) for other methods.

### 2. Write a function config

```yaml
# functions/fn-change-repo.yaml
apiVersion: builtin
kind: PatchTransformer
metadata:
  name: fn-change-repo
  annotations:
    config.karmafun.dev/cleanup: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
patch: |-
  - op: replace
    path: /spec/source/repoURL
    value: https://github.com/myname/autocloud.git
  - op: replace
    path: /spec/source/targetRevision
    value: feature/experiment
target:
  group: argoproj.io
  version: v1alpha1
  kind: Application
  annotationSelector: "autocloud/local-application=true"
```

### 3. Run

```console
kustomize fn run --enable-exec --fn-path functions applications
```

The `config.karmafun.dev/cleanup: "true"` annotation removes kustomize's
internal tracking annotations from the output (see
[Annotations](usage/annotations.md)).

## Related

- [kpt](https://kpt.dev/guides/rationale) — takes in-place transformation
  further with package versioning.
- [krmfnsops](https://github.com/kaweezle/krmfnsops) — the SOPS integration used
  by `SopsGenerator`.

<!-- reference links -->

[KRM Functions]:
  https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
