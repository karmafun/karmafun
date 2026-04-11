package process

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestBuilderFuncBuild(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	wantErr := errors.New("boom")

	b := builderFunc(func(id resid.Gvk) (resmap.Configurable, error) {
		req.Equal("TestKind", id.Kind)
		return nil, wantErr
	})

	_, err := b.Build(resid.Gvk{Kind: "TestKind"})
	req.ErrorIs(err, wantErr)
}

func TestAppendResources_ReturnsAnnotationTransferError(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	config, err := yaml.Parse(`apiVersion: builtin
kind: AnyGenerator
metadata:
  name: cfg
`)
	req.NoError(err)

	invalidGenerated, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata: invalid
`)
	req.NoError(err)

	rl := &framework.ResourceList{FunctionConfig: config}
	err = appendResources(rl, []*yaml.RNode{invalidGenerated}, map[string]string{"config.karmafun.dev/path": "x.yaml"})
	req.Error(err)
	req.Contains(err.Error(), "while transferring annotations")
}
