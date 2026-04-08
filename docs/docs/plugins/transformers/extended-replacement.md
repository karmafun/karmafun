<!-- cSpell: words argocd autocloud kustomize instana websecure citest myhost sishserver holepunch sshconfig bcrypt traefik -->

# ReplacementTransformer (extended)

karmafun's `ReplacementTransformer` extends the standard kustomize
[ReplacementTransformer](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/replacements/)
with three additional capabilities:

1. **Structured content paths** — navigate _inside_ string fields that contain
   embedded YAML, JSON, TOML, or INI data.
2. **Regex replacement** — replace a regex capture group within a string field.
3. **Encoding** — encode the source value (`base64`, `bcrypt`, `hex`) before
   writing it to the target.
4. **External source** — load replacement data from an external file or
   kustomization instead of injecting it into the pipeline.

## Basic usage

Same as the standard kustomize transformer:

```yaml
apiVersion: builtin
kind: ReplacementTransformer
metadata:
  name: replace-values
  annotations:
    config.karmafun.dev/cleanup: "true"
    config.karmafun.dev/prune-local: "true"
    config.kubernetes.io/function: |
      exec:
        path: karmafun
replacements:
  - source:
      kind: ConfigMap
      name: my-values
      fieldPath: data.repoURL
    targets:
      - select:
          kind: Application
        fieldPaths:
          - spec.source.repoURL
```

## Structured content paths

Prefix a path segment with `!!yaml.`, `!!json.`, `!!toml.`, or `!!ini.` to
navigate inside a string field that contains serialised data in that format.

The typical use case is an Argo CD `Application` with inline Helm values:

```yaml
# The application has:
#   spec.source.helm.values: |
#     ingressRoute:
#       dashboard:
#         enabled: false
```

To change `enabled` to `true`:

```yaml
replacements:
  - source:
      kind: LocalConfiguration
      fieldPath: data.traefik.dashboard_enabled
    targets:
      - select:
          kind: Application
          name: traefik
        fieldPaths:
          - spec.source.helm.values.!!yaml.ingressRoute.dashboard.enabled
```

## Array element matching

Reference an array element by a field value rather than by index:

```yaml
fieldPaths:
  - spec.source.helm.parameters.[name=common.repoURL].value
```

This survives array reordering, unlike a hardcoded index like
`spec.source.helm.parameters.1.value`.

## Regex replacement

Use `!!regex.<pattern>.<group>` to replace a capture group within a string field:

```yaml
fieldPaths:
  - data.config.!!regex.^\s+HostName\s+(\S+)\s*$.1
```

- `^\s+HostName\s+(\S+)\s*$` — the regular expression (the whole line is matched).
- `1` — the capture group number to replace.

Example — change the `HostName` line in an SSH config stored as a ConfigMap field:

```yaml
replacements:
  - source:
      kind: LocalConfiguration
      fieldPath: data.sish.server
    targets:
      - select:
          kind: ConfigMap
          name: sish-client
        fieldPaths:
          - data.config.!!regex.^\s+HostName\s+(\S+)\s*$.1
          - data.known_hosts.!!regex.^\[(\S+)\].1
```

## Encoding

Use the `options.encoding` field on the source to encode the value before
writing:

```yaml
replacements:
  - source:
      name: my-values
      fieldPath: data.admin_password
      options:
        encoding: base64
    targets:
      - select:
          kind: Secret
          name: argocd-secret
        fieldPaths:
          - data.admin.password
```

Supported encodings: `base64`, `bcrypt`, `hex`.

!!! note
    `bcrypt` generates a new hash on every run.

## External source file

Load replacement values from a file (or kustomization) instead of injecting
them into the pipeline:

```yaml
# properties.yaml  ← referenced, not injected
apiVersion: config.karmafun.dev/v1alpha1
kind: PlatformValues
metadata:
  name: platform-values
data:
  traefik:
    dashboard_enabled: true
```

```yaml
apiVersion: builtin
kind: ReplacementTransformer
metadata:
  name: replace-values
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
source: properties.yaml      # ← side-loaded; never enters the pipeline
replacements:
  - source:
      kind: PlatformValues
      fieldPath: data.traefik.dashboard_enabled
    targets:
      - select:
          kind: Application
          name: traefik
        fieldPaths:
          - spec.source.helm.values.!!yaml.ingressRoute.dashboard.enabled
```
