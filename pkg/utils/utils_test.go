package utils_test

// cSpell: words kioutil
import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/karmafun/karmafun/pkg/utils"
)

var emptyConfigAnnotationProperties = utils.AnnotationProperties{
	Path: ".karmafun.yaml",
}

func TestTransferAnnotationsToNode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		configAnnotations *utils.AnnotationProperties
		wantAnnotations   map[string]string
		name              string
		content           string
		index             int
		wantErr           bool
	}{
		{
			name: "When no annotations are set, only path annotation is added",
			content: `apiVersion: v1
kind: ConfigMap
metadata:
    name: test-configmap
data:
    key: value
`,
			configAnnotations: &emptyConfigAnnotationProperties,
			index:             3,
			wantErr:           false,
			wantAnnotations: map[string]string{
				kioutil.PathAnnotation:        ".karmafun.yaml",
				kioutil.LegacyPathAnnotation:  ".karmafun.yaml", //nolint:staticcheck // still in use.
				kioutil.IndexAnnotation:       "3",
				kioutil.LegacyIndexAnnotation: "3", //nolint:staticcheck // still in use.
			},
		},
		{
			name: "Setting path at the node level",
			content: `apiVersion: v1
kind: ConfigMap
metadata:
    name: test-configmap
    annotations:
        config.karmafun.dev/path: "custom-path.yaml"
data:
    key: value
`,
			configAnnotations: &emptyConfigAnnotationProperties,
			index:             0,
			wantErr:           false,
			wantAnnotations: map[string]string{
				kioutil.PathAnnotation:       "custom-path.yaml",
				kioutil.LegacyPathAnnotation: "custom-path.yaml", //nolint:staticcheck // still in use.
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)
			node, err := yaml.Parse(tt.content)
			req.NoError(err, "while parsing content")
			gotErr := utils.TransferAnnotationsToNode(node, tt.configAnnotations, tt.index)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("TransferAnnotationsToNode() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("TransferAnnotationsToNode() succeeded unexpectedly")
			}
			annotations := node.GetAnnotations()
			req.Equal(tt.wantAnnotations, annotations, "annotations after transfer mismatch")
		})
	}
}

func TestGetPathAnnotationProperties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		annotations map[string]string
		name        string
		want        utils.AnnotationProperties
		config      bool
	}{
		{
			name:        "No annotations set",
			annotations: map[string]string{},
			config:      true,
			want: utils.AnnotationProperties{
				Path:    ".karmafun.yaml",
				PathSet: false,
			},
		},
		{
			name: "Empty index means -1 and index set",
			annotations: map[string]string{
				utils.FunctionAnnotationIndex: "",
			},
			config: true,
			want: utils.AnnotationProperties{
				Path:     ".karmafun.yaml",
				PathSet:  false,
				Index:    -1,
				IndexSet: true,
			},
		},
		{
			name: "Other index invalid index means -1 and index set",
			annotations: map[string]string{
				utils.FunctionAnnotationIndex: "not_set",
			},
			config: false,
			want: utils.AnnotationProperties{
				Index:    -1,
				IndexSet: true,
			},
		},
		{
			name: "Path and index annotations set at config",
			annotations: map[string]string{
				utils.FunctionAnnotationPath:  "custom-path.yaml",
				utils.FunctionAnnotationIndex: "5",
			},
			config: true,
			want: utils.AnnotationProperties{
				Path:     "custom-path.yaml",
				PathSet:  true,
				Index:    5,
				IndexSet: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)
			got := utils.GeAnnotationProperties(tt.annotations, tt.config)
			req.Equal(tt.want, *got)
		})
	}
}
