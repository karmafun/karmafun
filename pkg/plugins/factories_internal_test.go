package plugins

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/resmap"
)

type fakeTransformer struct {
	configErr    error
	transformErr error
}

func (f *fakeTransformer) Config(_ *resmap.PluginHelpers, _ []byte) error {
	return f.configErr
}

func (f *fakeTransformer) Transform(_ resmap.ResMap) error {
	return f.transformErr
}

func TestMultiTransformer_Config(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfgErr  error
		name    string
		wantErr string
	}{
		{name: "success"},
		{name: "error", cfgErr: errors.New("bad cfg"), wantErr: "configuring transformer: bad cfg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)
			mt := &MultiTransformer{transformers: []resmap.TransformerPlugin{
				&fakeTransformer{configErr: tt.cfgErr},
			}}

			err := mt.Config(nil, []byte("x"))
			if tt.wantErr != "" {
				req.ErrorContains(err, tt.wantErr)
				return
			}
			req.NoError(err)
		})
	}
}

func TestMultiTransformer_Transform(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tfErr   error
		name    string
		wantErr string
	}{
		{name: "success"},
		{name: "error", tfErr: errors.New("bad transform"), wantErr: "transforming resources: bad transform"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)
			mt := &MultiTransformer{transformers: []resmap.TransformerPlugin{
				&fakeTransformer{transformErr: tt.tfErr},
			}}

			err := mt.Transform(nil)
			if tt.wantErr != "" {
				req.ErrorContains(err, tt.wantErr)
				return
			}
			req.NoError(err)
		})
	}
}
