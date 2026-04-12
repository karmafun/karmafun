package utils_test

// cSpell: words kioutil karmafun

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/utils"
)

func makeResourceFromYAML(t *testing.T, y string) *resource.Resource {
	t.Helper()
	node, err := yaml.Parse(y)
	if err != nil {
		t.Fatalf("parsing yaml: %v", err)
	}
	return &resource.Resource{RNode: *node}
}

func TestRemoveBuildAnnotations_RemovesAnnotations(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	r := makeResourceFromYAML(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    config.kubernetes.io/previousKinds: OldKind
    config.kubernetes.io/previousNames: old-name
    config.kubernetes.io/prefixes: prefix-
    config.kubernetes.io/suffixes: "-suffix"
    config.kubernetes.io/previousNamespaces: old-ns
    config.kubernetes.io/refBy: someone
    config.kubernetes.io/generatorBehavior: merge
    config.kubernetes.io/needsHashSuffix: "true"
    custom.annotation/keep: value
`)
	utils.RemoveBuildAnnotations(r)

	got := r.GetAnnotations()
	req.NotContains(got, utils.BuildAnnotationPreviousKinds)
	req.NotContains(got, utils.BuildAnnotationPreviousNames)
	req.NotContains(got, utils.BuildAnnotationPrefixes)
	req.NotContains(got, utils.BuildAnnotationSuffixes)
	req.NotContains(got, utils.BuildAnnotationPreviousNamespaces)
	req.NotContains(got, utils.BuildAnnotationsRefBy)
	req.NotContains(got, utils.BuildAnnotationsGenBehavior)
	req.NotContains(got, utils.BuildAnnotationsGenAddHashSuffix)
	// Custom annotation should be preserved
	req.Equal("value", got["custom.annotation/keep"])
}

func TestRemoveBuildAnnotations_EmptyAnnotations(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	r := makeResourceFromYAML(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`)
	// Should not panic with empty annotations
	utils.RemoveBuildAnnotations(r)
	req.Empty(r.GetAnnotations())
}

func TestTransferAnnotations_MultipleResources(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	configAnnotations := map[string]string{
		utils.FunctionAnnotationPath:  ".output.yaml",
		utils.FunctionAnnotationIndex: "0",
	}

	node1, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key: value1
`)
	req.NoError(err)

	node2, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
data:
  key: value2
`)
	req.NoError(err)

	nodes := []*yaml.RNode{node1, node2}
	err = utils.TransferAnnotations(nodes, configAnnotations)
	req.NoError(err)
}

func TestTransferAnnotations_EmptyList(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	configAnnotations := map[string]string{
		utils.FunctionAnnotationPath: ".output.yaml",
	}

	err := utils.TransferAnnotations([]*yaml.RNode{}, configAnnotations)
	req.NoError(err)
}

func TestTransferAnnotations_WithLocalAndInjectLocal(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	configAnnotations := map[string]string{
		utils.FunctionAnnotationLocalConfig: utils.TrueValue,
		utils.FunctionAnnotationInjectLocal: utils.TrueValue,
		utils.FunctionAnnotationKind:        "Secret",
		utils.FunctionAnnotationApiVersion:  "v1",
		utils.FunctionAnnotationPath:        "output.yaml",
	}

	node, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`)
	req.NoError(err)

	err = utils.TransferAnnotations([]*yaml.RNode{node}, configAnnotations)
	req.NoError(err)

	// After transfer, the kind should be modified since InjectLocal was set
	annotations := node.GetAnnotations()
	// inject-local annotation should be removed
	req.NotContains(annotations, utils.FunctionAnnotationInjectLocal)
}

func TestUnLocal_FiltersLocalResources(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	localNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: local-cm
  annotations:
    config.karmafun.dev/local-config: "true"
data:
  key: value
`)
	req.NoError(err)

	normalNode, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: normal-cm
data:
  key: value
`)
	req.NoError(err)

	nodes := []*yaml.RNode{localNode, normalNode}
	result, err := utils.UnLocal(nodes)
	req.NoError(err)
	req.Len(result, 1, "local-config resource should be filtered out")
	meta, err := result[0].GetMeta()
	req.NoError(err)
	req.Equal("normal-cm", meta.Name)
}

func TestUnLocal_CopiesPathAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	node, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  annotations:
    config.karmafun.dev/path: "custom-path.yaml"
data:
  key: value
`)
	req.NoError(err)

	nodes := []*yaml.RNode{node}
	result, err := utils.UnLocal(nodes)
	req.NoError(err)
	req.Len(result, 1)

	annotations := result[0].GetAnnotations()
	req.Equal("custom-path.yaml", annotations["config.kubernetes.io/path"])
	req.NotContains(annotations, utils.FunctionAnnotationPath)
}

func TestUnLocal_CopiesIndexAnnotation(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	node, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  annotations:
    config.karmafun.dev/path: "custom-path.yaml"
    config.karmafun.dev/index: "5"
data:
  key: value
`)
	req.NoError(err)

	nodes := []*yaml.RNode{node}
	result, err := utils.UnLocal(nodes)
	req.NoError(err)
	req.Len(result, 1)

	annotations := result[0].GetAnnotations()
	req.Equal("5", annotations["config.kubernetes.io/index"])
	req.NotContains(annotations, utils.FunctionAnnotationIndex)
}

func TestResourceMapFromNodes(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	node1, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key: value1
`)
	req.NoError(err)

	node2, err := yaml.Parse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
data:
  key: value2
`)
	req.NoError(err)

	rm := utils.ResourceMapFromNodes([]*yaml.RNode{node1, node2})
	req.NotNil(rm)
	req.Equal(2, rm.Size())
}

func TestResourceMapFromNodes_Empty(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	rm := utils.ResourceMapFromNodes([]*yaml.RNode{})
	req.NotNil(rm)
	req.Equal(0, rm.Size())
}

func TestConcat(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	seq1 := func(yield func(int) bool) {
		for _, v := range []int{1, 2, 3} {
			if !yield(v) {
				return
			}
		}
	}
	seq2 := func(yield func(int) bool) {
		for _, v := range []int{4, 5, 6} {
			if !yield(v) {
				return
			}
		}
	}

	result := slices.Collect(utils.Concat(seq1, seq2))
	req.Equal([]int{1, 2, 3, 4, 5, 6}, result)
}

func TestConcat_EarlyStop(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	seq1 := func(yield func(int) bool) {
		for _, v := range []int{1, 2, 3} {
			if !yield(v) {
				return
			}
		}
	}
	seq2 := func(yield func(int) bool) {
		for _, v := range []int{4, 5, 6} {
			if !yield(v) {
				return
			}
		}
	}

	// Take only first 2 elements
	result := make([]int, 0, 2)
	for v := range utils.Concat(seq1, seq2) {
		result = append(result, v)
		if len(result) == 2 {
			break
		}
	}
	req.Equal([]int{1, 2}, result)
}

func TestConcat_Empty(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result := slices.Collect(utils.Concat[int]())
	req.Empty(result)
}
