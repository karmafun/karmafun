# Karmafun Development Guide for AI Agents

<!-- cSpell: words builtinplugintype stringifier containerizable nfpms -->

## Project Overview

Karmafun is a
[kustomize plugin](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/)
providing a set of KRM (Kubernetes Resource Model) Functions for in-place
transformations in kustomize projects. It wraps the kustomize framework to
expose additional generators and transformers beyond kustomize's built-ins.

### Core Architecture

Karmafun acts as a KRM function container that processes ResourceLists through
the **kustomize KRM Framework**
([kyaml/fn/framework](https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/fn/framework)):

```
Input YAML (FunctionConfig + Items)
    ↓
processor.Process(ResourceList)
    ↓
MakeBuiltinPlugin(GVK) → Generator or Transformer
    ↓
plugin.Config(helpers, yaml) + plugin.Transform/Generate()
    ↓
Output YAML (modified Items)
```

**Key Entry Point:** [main.go](main.go) - processor implements
`framework.ResourceListProcessor`

### Project Structure

- **`main.go`** - Entry point: KRM function processor dispatching to
  generators/transformers
- **`pkg/plugins/`** - Plugin factory system mapping GVKs to implementations
  - `factories.go` - `MakeBuiltinPlugin()`, `TransformerFactories`,
    `GeneratorFactories` maps
  - `builtinplugintype_string.go` - Generated enum stringifier (do not edit)
- **`pkg/extras/`** - Custom generators and transformers
  - **Generators**: `GitConfigMapGenerator`, `KustomizationGenerator`,
    `SopsGenerator`
  - **Transformers**: `ExtendedReplacementTransformer`, `RemoveTransformer`,
    `Extender` (yaml/json/toml/ini support)
- **`pkg/utils/`** - Shared utilities
  - `constants.go` - Karmafun annotation domains (`config.karmafun.dev`,
    `config.kubernetes.io`)
  - `utils.go` - ResourceMap/RNode helpers, annotation handling

### Plugin System Pattern

All plugins implement the kustomize `resmap.Configurable` interface:

```go
type MyPlugin struct {
    h *resmap.PluginHelpers // Set by framework during Config()
    // Plugin-specific fields from YAML unmarshal
}

func (p *MyPlugin) Config(h *resmap.PluginHelpers, c []byte) error {
    // 1. Unmarshal YAML config into p's fields
    // 2. Store helpers for later use (Generate/Transform needs it)
    p.h = h
    return nil
}

func (p *MyPlugin) Generate() (resmap.ResMap, error) { /*for Generators*/ }
func (p *MyPlugin) Transform(m resmap.ResMap) error   { /*for Transformers*/ }

func NewMyGeneratorPlugin() resmap.GeneratorPlugin   { return &MyPlugin{} }
func NewMyTransformerPlugin() resmap.TransformerPlugin { return &MyPlugin{} }
```

**Critical:** Plugins are instantiated fresh for each kustomize invocation.
Store mutable state in plugin fields during `Config()`.

### Annotation-Driven Behavior

Karmafun uses special annotations to control plugin execution:

| Annotation                         | Meaning                                                   | Example Use                      |
| ---------------------------------- | --------------------------------------------------------- | -------------------------------- |
| `config.kubernetes.io/function`    | Marks KRM function (kustomize framework)                  | See kustomize docs               |
| `config.karmafun.dev/inject-local` | Inject FunctionConfig as resource in output               | For non-plugin configs           |
| `config.karmafun.dev/cleanup`      | Remove build annotations from transformed resources       | After ReplacementTransformer     |
| `config.karmafun.dev/prune-local`  | Remove all local-config marked resources before returning | Final pruning step               |
| `config.karmafun.dev/kind`         | Override Kind for generated resources (SopsGenerator)     | `kind: Secret` → `SopsGenerator` |
| `config.karmafun.dev/apiVersion`   | Override ApiVersion for generated resources               | Needed for custom resource types |

See [pkg/utils/constants.go](../pkg/utils/constants.go) for full list.

## Key Components Deep Dive

### Generator Plugins

**GitConfigMapGenerator** - Introspects local git repo:

- Auto-populates ConfigMap with `repoURL` (from git remote, default "origin")
- Auto-populates with `targetRevision` (current branch name)
- Useful for Argo CD app customization (capture git state at kustomization time)

**KustomizationGenerator** - Runs nested kustomizations:

- Invokes `kubectl kustomize <directory>` on specified directory
- Outputs flattened ResourceMap from that kustomization

**SopsGenerator** - Decrypts SOPS-encrypted files:

- Supports encrypted YAML/JSON files via
  [github.com/getsops/sops](https://github.com/getsops/sops)
- Two modes: `files:` (list of encrypted files) or inline `sops:` block
- Sets `config.karmafun.dev/inject-local=true` on decrypted resources

### Transformer Plugins

**ExtendedReplacementTransformer** - Field replacement with embedded data
support:

- Extends kustomize's ReplacementTransformer to handle nested structures
- Supports `!!json.`, `!!yaml.`, `!!toml.`, `!!ini.` prefixes for embedded paths
- Example: `!!json.common.targetRevision` targets
  `"common": {"targetRevision": "..."}`
- Supports `regex!!` prefix for regex-based replacements. Ex:
  `!!regex.^\s+HostName\s+(\S+)\s*$.1` replaces `target.link` in
  `HostName target.link`
- Supports complex array element matching. Ex:
  `spec.source.helm.parameters.[name=common.repoURL].value`.
- See [Extender interface](../pkg/extras/extender.go) for implementation

**RemoveTransformer** - Removes resources by selector:

- Like inverse of builtin transformers
- Configured with `targets:` list of selectors
- Useful for removing generated or temporary resources before output

## Development Workflows

### Build and Test

```bash
# Unit tests
go test ./...

# Tests with verbose output and coverage
go test -v -race -covermode=atomic -coverprofile=coverage.out ./...

# Linting and formatting (uses golangci-lint)
golangci-lint run --fix

# Single-target build (linux/amd64, APK format)
goreleaser build --single-target --auto-snapshot --clean

# Multi-platform build (darwin, linux, windows)
goreleaser --auto-snapshot --skip=publish --clean
```

### Testing Patterns

Since karmafun interacts with kustomize's internal APIs:

1. **Plugin testing**: Create test YAML configs, unmarshal into plugin struct,
   call `Config(helpers)` then `Transform/Generate()`
2. **ResourceMap operations**: Use `utils.ResourceMapFromNodes()` to convert
   YAML node lists
3. **Mock files**: Use in-memory filesystem or fixtures in `tests/` directory
4. **Duplicate test detection**: golangci-lint reports duplicate tests - extract
   to helper functions

### Integration Testing

In `tests/` directory, each subdirectory contains the following subdirectories:

- `original/` - Input YAML resources (Items)
- `functions/` - FunctionConfig YAMLs (one or more) to apply in lexical order
- `expected/` - Expected output YAML resources after applying functions

Run integration tests with:

```bash
# Test with kustomize fn run (requires kustomize installed)
go build -o karmafun && ./tests/test_karmafun.sh

# Test with kpt (requires kpt installed)
go build -o karmafun && ./tests/test_karmafun_kpt.sh
```

For kustomize:

- The `original/` resources are copied to `applications/` directory.
- All FunctionConfig in `functions/` are applied at once via `kustomize fn run`.
- results are compared to `expected/` resources via on ouput normalized via `yq`
  to ignore ordering and formatting differences.

For kpt:

- The `original/` resources are merged into a single temporary file via
  `kpt source`.
- Each FunctionConfig in `functions/` is applied sequentially via `kpt fn eval`.
- Results are split via `kpt sink` into separate resources in `applications/`
  and compared to `expected/` resources.

### Adding a New Transformer

1. Create `pkg/extras/MyTransformer.go` implementing `resmap.TransformerPlugin`
2. Add to [TransformerFactories map](../pkg/plugins/factories.go#L104)
3. Add to `BuiltinPluginType` enum in
   [factories.go](../pkg/plugins/factories.go#L23)
4. Generate stringer: `go generate ./pkg/plugins/`
5. Test via `kustomize fn run` with your YAML config

Example GVK detection:

```yaml
apiVersion: builtin
kind: MyTransformer # Auto-detected from FunctionConfig
metadata:
  name: example
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: karmafun
# Plugin config fields:
someField: value
```

## Project-Specific Conventions

### Error Handling

- Wrap errors with context: `fmt.Errorf("action subject: %w", err)`
- Use `errors.WrapPrefixf()` (kyaml) in some generators for consistency
- Plugin Config() errors bubble up through framework

### ResourceList Processing

The processor receives a `*framework.ResourceList`:

- `FunctionConfig` (RNode) - plugin configuration (first item in YAML)
- `Items` ([]\*yaml.RNode) - resources to transform/generate
- Modify `Items` in-place; framework writes output

Key pattern from [main.go](main.go#L75):

```go
switch v := plugin.(type) {
case resmap.Transformer:
    transformResources(rl, v, configAnnotations) // Modifies rl.Items
case resmap.Generator:
    generateResources(rl, v, configAnnotations)  // Appends to rl.Items
}
```

### Annotation Inheritance

When appending generated resources, transfer annotations from FunctionConfig:

```go
utils.TransferAnnotations(generatedNodes, configAnnotations)
```

Supports propagating `config.karmafun.dev/*` directives to outputs.

## Common Pitfalls

1. **Plugin state management**: Don't store mutable state in package-level vars.
   All state goes in plugin struct fields.
2. **ResourceMap ordering**: Order matters for kustomize output; use stable
   sorts when manipulating maps.
3. **Embedded data access**: `Extender` interface requires correct encoding
   prefix (`!!json.`, `!!yaml.`, etc.) - will fail silently otherwise.
4. **Generator vs Transformer confusion**:
   - Generators: Create new resources (increase Items count)
   - Transformers: Modify existing resources (same Items count)
5. **Annotation cleanup**: Use `config.karmafun.dev/cleanup` annotation to
   remove kustomize build annotations before output (required for some use
   cases).

## CI/CD

- **Tests**: Run on every push/PR via GitHub Actions
- **Releases**: Triggered on `v*` tags → goreleaser builds APK+binary to dist/
- **APK distribution**: Depends on goreleaser `.nfpms` config

## Architecture Decision Records

### Why KRM Framework vs Standalone Binary?

KRM (Kubernetes Resource Model) plugin architecture allows:

- Composable transformations (chain multiple generators/transformers)
- Direct kustomize integration via `kustomize fn run`
- YAML-native configuration (no CLI flags)
- Containerizable execution

### Why Factory Pattern for Plugins?

Plugins are instantiated on-demand from GVK (Group/Version/Kind) in
FunctionConfig:

- Supports both kustomize built-in plugins and custom extensions
- Allows fallback to local injection if plugin not found
  (`FunctionAnnotationInjectLocal`)
- Efficient: Only instantiate plugins actually used

## Quick Reference

| Task               | Command                                                               |
| ------------------ | --------------------------------------------------------------------- |
| Run all tests      | `go test ./...`                                                       |
| Test with coverage | `go test -v -race -covermode=atomic -coverprofile=coverage.out ./...` |
| Build APK/binary   | `goreleaser build --single-target --auto-snapshot --clean`            |
| Lint and fix       | `golangci-lint run --fix`                                             |
| Generate code      | `go generate ./pkg/plugins/`                                          |
| View plugin types  | `pkg/plugins/builtinplugintype_string.go`                             |
