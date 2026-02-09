package extras_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/kio"

	"github.com/karmafun/karmafun/pkg/extras"
	"github.com/karmafun/karmafun/pkg/plugins"
)

const configmapYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-configmap
data:
  key: value
`

func TestKCLGeneratorPlugin_Generate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string // description of this test case
		functionConfig string // the functionConfig for the plugin
		want           string // the expected output yaml string
		wantErr        bool
	}{
		{
			name: "generate a configmap with KCL code",
			functionConfig: `
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: example-kcl-run
spec:
  source: |
    {
      apiVersion = "v1"
      kind = "ConfigMap"
      metadata.name = "example-configmap"
      data.key = "value"
    }
`,
			want:    configmapYaml,
			wantErr: false,
		},
		{
			name: "Configmap with source file",
			functionConfig: `
apiVersion: kcl.dev/v1alpha1
kind: KCLRun
metadata:
  name: example-kcl-run
spec:
  source: ../../tests/kcl/configmap.k
`,
			want:    configmapYaml,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			helpers, err := plugins.NewPluginHelpers()
			req.NoError(err, "creating plugin helpers should not error")

			// TODO: construct the receiver type.
			var p extras.KCLGeneratorPlugin
			err = p.Config(helpers, []byte(tt.functionConfig))
			req.NoError(err, "configuring plugin should not error")
			got, gotErr := p.Generate()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Generate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Generate() succeeded unexpectedly")
			}
			var b bytes.Buffer
			req.NoError(err, "encoding generated resources should not error")
			err = kio.ByteWriter{Writer: &b}.Write(got.ToRNodeSlice())
			req.NoError(err, "writing generated resources should not error")
			req.Equal(tt.want, b.String(), "generated resources should match expected output")
		})
	}
}
