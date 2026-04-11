<!-- cSpell: words stdlib oldname newname filesys -->

# Refactorings Needed For Uncovered Paths

Goal: cover branches that currently depend on direct stdlib or external calls
that tests cannot control without invasive environment manipulation.

## plugins/factories.go

- `GetPluginPath`: inject `userConfigDir func() (string, error)` because
  `os.UserConfigDir` failure branch not controllable in stable unit tests.
- `CreatePluginDirectoryHierarchy`: inject executable path provider
  (`executablePath func() (string, error)`) because `os.Executable` failure
  branch not controllable.
- `CreatePluginDirectoryHierarchy`: inject symlink creator
  (`symlink func(oldname, newname string) error`) because `os.Symlink` failure
  branch only reproducible with host-specific permissions/filesystem behavior.
- `CreatePluginDirectoryHierarchy`: inject plugin kind iterator or factories map
  copy when deterministic ordering assertions needed in tests.
- `RemovePluginDirectoryHierarchy`: share same path provider abstraction used by
  `GetPluginPath` to force path resolution failures in tests.

## cmd/build/build_cmd.go

- `SplitResMapToDir`: inject YAML marshal function
  (`marshalResource func(any) ([]byte, error)`) because `yaml.Marshal(resource)`
  error branch is hard to trigger with valid kustomize resources.
- `runBuildCommand`: inject kustomization runner
  (`runKustomizations func(fs filesys.FileSystem, dir string) (resmap.ResMap, error)`)
  to test all error branches without relying on full kustomize runtime behavior.

## process/process.go

- `Process`: inject plugin helper factory
  (`newPluginHelpers func() (*resmap.PluginHelpers, error)`) because
  `plugins.NewPluginHelpers` failure branch currently depends on
  global/environment/runtime details.

## main.go

- `main`: split command construction and execution into injectable functions
  (`buildRootCommand`, `executeCommand`) to unit test exit/error branches
  without process exit side effects.
- `main`: replace direct `os.Exit` with injectable exit function in runtime
  struct for deterministic tests.

## hack/nocov.go

- Script-style command currently binds many direct filesystem and parser calls.
  Wrap dependencies in interfaces (`fs`, `coverParser`, `writer`, `argSource`)
  to make branch coverage realistic.

## Minimal implementation pattern

1. Define small dependency structs next to each command/processor.
2. Keep default constructors wiring real stdlib functions.
3. Expose package-private constructors for tests with fake dependencies.
4. Preserve public API and CLI behavior.
