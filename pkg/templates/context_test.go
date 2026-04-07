package templates_test

// cSpell: words filesys testify karmafun gotmpl
import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/karmafun/karmafun/pkg/templates"
)

func TestMergeMaps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    map[string]any
		b    map[string]any
		want map[string]any
	}{
		{
			name: "Empty maps return empty map",
			a:    map[string]any{},
			b:    map[string]any{},
			want: map[string]any{},
		},
		{
			name: "B values override A values",
			a:    map[string]any{"key": "valueA"},
			b:    map[string]any{"key": "valueB"},
			want: map[string]any{"key": "valueB"},
		},
		{
			name: "Nil value in B skips the key",
			a:    map[string]any{"key": "valueA"},
			b:    map[string]any{"key": nil},
			want: map[string]any{"key": "valueA"},
		},
		{
			name: "Keys only in A are preserved",
			a:    map[string]any{"onlyA": "value"},
			b:    map[string]any{},
			want: map[string]any{"onlyA": "value"},
		},
		{
			name: "Keys only in B are added",
			a:    map[string]any{},
			b:    map[string]any{"onlyB": "value"},
			want: map[string]any{"onlyB": "value"},
		},
		{
			name: "Nested maps are merged recursively",
			a: map[string]any{
				"nested": map[string]any{
					"fromA": "valueA",
					"both":  "fromA",
				},
			},
			b: map[string]any{
				"nested": map[string]any{
					"fromB": "valueB",
					"both":  "fromB",
				},
			},
			want: map[string]any{
				"nested": map[string]any{
					"fromA": "valueA",
					"fromB": "valueB",
					"both":  "fromB",
				},
			},
		},
		{
			name: "B non-map replaces A map",
			a:    map[string]any{"key": map[string]any{"nested": "value"}},
			b:    map[string]any{"key": "scalar"},
			want: map[string]any{"key": "scalar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := templates.MergeMaps(tt.a, tt.b)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewPlatformValues(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	req.NotNil(pv)
	req.Equal("config.karmafun.dev/v1alpha1", pv.APIVersion)
	req.Equal("PlatformValues", pv.Kind)
	req.Equal("platform-values", pv.Name)
	req.NotNil(pv.Data)
	req.Empty(pv.Data)
}

func TestPlatformValues_AsMap(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	pv.Data = map[string]any{
		"key": "value",
		"nested": map[string]any{
			"inner": "inner-value",
		},
	}
	m := pv.AsMap()
	req.Contains(m, "data")
	data, ok := m["data"].(map[string]any)
	req.True(ok, "data should be map[string]any")
	req.Equal("value", data["key"])
	nested, ok := data["nested"].(map[string]any)
	req.True(ok, "nested should be map[string]any")
	req.Equal("inner-value", nested["inner"])
}

func TestReadPlatformValues_FileNotFound(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	pv, err := templates.ReadPlatformValues(fs, "values.yaml")
	req.NoError(err)
	req.NotNil(pv)
	req.Empty(pv.Data, "should return empty values when file not found")
}

func TestReadPlatformValues_ValidFile(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	content := `apiVersion: config.karmafun.dev/v1alpha1
kind: PlatformValues
metadata:
  name: test-values
data:
  domain: example.com
  project:
    name: my-project
`
	err := fs.WriteFile("values.yaml", []byte(content))
	req.NoError(err)

	pv, err := templates.ReadPlatformValues(fs, "values.yaml")
	req.NoError(err)
	req.NotNil(pv)
	req.Equal("example.com", pv.Data["domain"])
	projectData, ok := pv.Data["project"].(map[string]any)
	req.True(ok, "project should be map[string]any")
	req.Equal("my-project", projectData["name"])
}

func TestReadPlatformValues_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	err := fs.WriteFile("values.yaml", []byte("not: valid: yaml: {"))
	req.NoError(err)

	_, err = templates.ReadPlatformValues(fs, "values.yaml")
	req.Error(err)
	req.Contains(err.Error(), "values.yaml")
}

func TestMergeValues_BothNonNil(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	pv.Data = map[string]any{
		"fromPlatform": "platformValue",
		"shared":       "platformShared",
	}
	sv := templates.NewPlatformValues()
	sv.Data = map[string]any{
		"fromSecrets": "secretValue",
		"shared":      "secretShared",
	}

	merged := templates.MergeValues(pv, sv)
	req.NotNil(merged)
	req.Equal("platformValue", merged.Data["fromPlatform"])
	req.Equal("secretValue", merged.Data["fromSecrets"])
	req.Equal("secretShared", merged.Data["shared"], "secrets should override platform values")
}

func TestMergeValues_NilPlatform(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	sv := templates.NewPlatformValues()
	sv.Data = map[string]any{"key": "value"}
	merged := templates.MergeValues(nil, sv)
	req.Equal(sv, merged)
}

func TestMergeValues_NilSecrets(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	pv := templates.NewPlatformValues()
	pv.Data = map[string]any{"key": "value"}
	merged := templates.MergeValues(pv, nil)
	req.Equal(pv, merged)
}

func TestReadSecretsValues_ReadSecrets(t *testing.T) {
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	content := testSecretsEncryptedWithData
	err := fs.WriteFile("secrets.sops.yaml", []byte(content))
	req.NoError(err)
	t.Setenv("SOPS_AGE_KEY", testSecretAgeKey)

	values, err := templates.ReadSecretsValues(fs, "secrets.sops.yaml")
	req.NoError(err)
	req.NotNil(values)
	req.Equal("karmafun-secrets", values.Name)
	valuesMap := values.AsMap()
	req.Contains(valuesMap, "data")
	data, ok := valuesMap["data"].(map[string]any)
	req.True(ok, "data should be map[string]any")
	req.Contains(data, "argocd")
	argocdData, ok := data["argocd"].(map[string]any)
	req.True(ok, "argocd should be map[string]any")
	req.Contains(argocdData, "admin_password")
	req.Equal("$2a$10$xdlX460lf/WbJNZU5bBoROj6U7oKgPbEcBrnXaemA6gsCzrAJtQ3y", argocdData["admin_password"])
}

func TestReadSecretsValues_FileNotFound(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	values, err := templates.ReadSecretsValues(fs, "nonexistent.sops.yaml")
	req.NoError(err)
	req.NotNil(values)
	req.Empty(values.Data, "should return empty values when secrets file not found")
}

func TestReadSecretsValues_InvalidYAML(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	fs := filesys.MakeFsInMemory()
	err := fs.WriteFile("secrets.sops.yaml", []byte("not: valid: yaml: {"))
	req.NoError(err)

	_, err = templates.ReadSecretsValues(fs, "secrets.sops.yaml")
	req.Error(err)
	req.Contains(err.Error(), "secrets.sops.yaml")
}

// cSpell: disable
// Regenerate fixture with:
// sops --config <(echo "creation_rules:\n  - encrypted_regex: ^data\$") -e -a 'age166k86d56ejs2ydvaxv2x3vl3wajny6l52dlkncf2k58vztnlecjs0g5jqq' plain.yaml | cat.
//
//nolint:lll,gosec // static fixture with long encrypted values
const testSecretsEncryptedWithData = `# Use:
#   sops -e secrets.dec.sops.yaml > secrets.sops.yaml
# to encrypt this file.
# The sample private key to use is in the data.sops.age_key.txt field of this file.
# Put in inside the ~/.config/sops/age/keys.txt file.
# Sample command to do that:
#  gojq -r --yaml-input '.data.sops["age_key.txt"]' ./samples/kustomization/secrets.dec.sops.yaml \
#    >> ~/.config/sops/age/keys.txt
# cSpell: disable
apiVersion: karmafun.dev/v1alpha1
kind: SopsGenerator
metadata:
    name: karmafun-secrets
data:
    argocd:
        admin_password: ENC[AES256_GCM,data:2oPFj6hjNdxYZO5aO7dSVwP9jc6oioAWImqeNaRTxkPlAZXfHsbSDJR5wYb093KyZoGWNZpbGC6W95CH,iv:V9nAJM55BjkBtK9biC/xY4h+M0smDVYZKzQlic7OKxc=,tag:nXfNxWyPY+hSNsuOuq5MUg==,type:str]
    chisel:
        AUTH: ENC[AES256_GCM,data:O2kQ0DK/Slt1WEs3aA==,iv:fTKTWI5SWZhmSrLsQyjdjHB+eCSwBL5I02mXyzBrc4I=,tag:p/i7dCigQ+ktRL9P/ZrB+A==,type:str]
    cloudflare:
        apiKey: ENC[AES256_GCM,data:Na6I/8kFSRmADbUmok/t5D/aif4HZPQzSwyubs+3HwuF4MgRjGGfuA==,iv:Zp4S2/hFFmECK/eKrULEniDnVyLKzafbN5QCAzeHQ2Y=,tag:8Bm0Lx1d50ORJVUez/gp3A==,type:str]
        credentials.json: ENC[AES256_GCM,data:6yx7gx241yQjtJJGI2Qxq86hjyinEYkcLcPBLNtbWzLYKSaQamgOr/CE4pCvx44i7M/U9kncSMuQAWX+bfQiComkJ8FryyHSx1aRKJxRiVo4hCmbIncMlzMFjwYk46aOr+FOgC5D2NYSenzlNJHu1GOGYaLGvPM1ak4et0Pn5aG7X6mMBHkgtOVoyyyb69IMlYbBqztimvs0c1hCyXMFRno0gxsYYA==,iv:ApHNsQ6xI69iuiy6OOde6NvBRmGGtrUi+dFnkmmLVAM=,tag:skerYq/9BIUspntaJU/A7Q==,type:str]
    github:
        oidc_client_secret: ENC[AES256_GCM,data:j8LKg63N5TlEmh4jvDoiqvJEcBB1wNjK9ytKchlUxlCdoYEFVQygIA==,iv:6SOsJXs6+D8NuyqKT6LMALw3IRazMGAIbPgU4tIc5DU=,tag:fmc0w9AQ+3uH5dFUCjPOiQ==,type:str]
        password: ENC[AES256_GCM,data:eQtAt0HROyKfzaspS4XFn/DAsSC/xQHdfKnCPbfY49j84/wvj8wHRA==,iv:5MiPCITU89x4nsczvgYt+QvgjqLE0/YEEfTcykfmdwY=,tag:yTB0Qs0c18GDekEy/enjIA==,type:str]
        ssh_key: ENC[AES256_GCM,data:d8X4kTmQA2nzfDGSavz1I5w8JKptoVHoBXWBaG4iDshRM+XMdhBDgXc22jHnOllKGjKNiAmr/Z59pXWf0FoRhrcbsI7k91BeDsdMpF7u38E8Dc4aMPuZZFRGFAzHhjnS/iYj7ODuuP2oEGePE9FA3cOuGXIjwr+bT4YbkwFIcOFIiTl1iAatnr8MRVvLyFRhcXY+STmZ0gUCUbWxivlNDobOwY95oZo0qy1POSGQfYGmX5FuAYy/Sy5ucmtNXauqY63r90+UVxphLCFGjK7L92qL87MFh7nABnQ4hUai3Rx0Y5FCMFgXWT8hNYIWXvLPFqfQD7lNAR//W5pKIFhLu5o5D74LXv+lnLBliA/gFXZv/cVX09OoISVcDc/e1WSVZp9uCwCdxd8XEvoiHJztIRi/sz9DWTzgOLX+URMSZ2OmtrHs5DyH8jdx2aS35AkwQ3VAFwlEJvWZKBy3VzxowgJvZETs7EWvkFmBRev0xShb3wnsV+oZifcHs/jrMBAp2hfPKKYoqPMGcFmAv9RVUrfgxTMTHEtIZzwR,iv:zkzWej8ZjBM0izeuJ3sDV3+3ZuqYztDBdAfQFj4vf/8=,tag:qmcmUU988xsUNXgZdytNQA==,type:str]
        webhook_secret: ENC[AES256_GCM,data:AYBD47Xa5l5+hwlLz6YwN38miOvjMn8zbLgIlc1Rngg=,iv:2svyA7ML1Eizi932HDPhFEcnngWIi3oJuOIiPRs8MZo=,tag:O78Ccn5fHSaP6G0WsGiTAg==,type:str]
    ovh:
        application_secret: ENC[AES256_GCM,data:PBHzXyKMyGTIP6hmSdpEK2BEtBOEma/rg+CKFK8NbBM=,iv:gxdj5K61lgorNFUp4NcS3JLY3wkT/JJ3PzbP/iowt3c=,tag:hezQcmW8tLNAdbSUsSFHAA==,type:str]
        consumer_key: ENC[AES256_GCM,data:YGcLgibuW3y351PPveVjlyTwCdveUZDKUxXekB6su9U=,iv:qWqaFSLEOtze1ZjJPDj1SKaHYjgafq1Jmck6CZ5S7jU=,tag:slKUfPUFOMSHyfein+DZrQ==,type:str]
    sops:
        age_key.txt: ENC[AES256_GCM,data:49sJsfG/70epz+xaph7e+kOCu3CfSIO2KV9JKLPfk8w5CosG11rP3kKxfcpJmeU7tqWj1pbcavh+dZOMbanyfHHp6Qay/0+gFfybkpIn90EUXMKaVhPiLnpEvq/b/3A69UVWFnm/Q0M762dwI06WM/6pCsEgCM67Q5H9PTPUa1NYMIKSgiCiQMcXLJmUh2R49bQ903UTH9sqFEorMvVL88fJmHweirwG4xYfHK3HuZmq65A3wFN17Q==,iv:eQi03p+kja50gZvW6gQyXRgqA7AmmkBq2Nyatk1qPwY=,tag:XMuoGqIbCWiTKJXmKnFd9Q==,type:str]
sops:
    age:
        - recipient: age166k86d56ejs2ydvaxv2x3vl3wajny6l52dlkncf2k58vztnlecjs0g5jqq
          enc: |
            -----BEGIN AGE ENCRYPTED FILE-----
            YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBYTkVKY21tNEJCb0g4QWRv
            eTR4dWZNZTlaRHJCNjFRbGFhTVAyd09YMWl3CkQ3VE1QeGFNTms4bWVSb1dYZ1ly
            UHowOFVRWUxTd2hDM2wrTkdNTWxqeWcKLS0tIE5iWE5aLzZROFlzYjNzUGozSnRK
            QndsNEVmZU43cnhlV3ZlRWpwdTFJSFkKGbNb1uQZeV1R0p0FabM6CfQ8twKsv/GJ
            uZPmP27zOb1MKrQ+SX4VGKF14vrswivpXlnyIp3Vaa2tcMAhtmWO3g==
            -----END AGE ENCRYPTED FILE-----
    lastmodified: "2026-04-08T06:43:50Z"
    mac: ENC[AES256_GCM,data:JEw1xkBvEsDA2TxMJByJAH/lUDK8azk8s2oLDnBCNp+JcFpTljN0oDtFks6QNOQ51qZtf5NGzr2WUYr4vs4ZeRPRZO8KjnqauSxWSP5Ypc7F1gpyJAUytWR4jer7xvxqU8m9yvbTK6pVe1VzG+cAO2nLLcMRJestcY3845TxFGg=,iv:UkbTbtO6hS2mKlCZLU+Y4O9vpzViFPZVUa476UVrhJU=,tag:FzlmZG+eJI3N9lk8+EAVSg==,type:str]
    encrypted_regex: ^data$
    version: 3.12.1`

//nolint:gosec // Private sample key for testing, no security risk.
const testSecretAgeKey = `# created: 2023-01-19T19:41:45Z
# public key: age166k86d56ejs2ydvaxv2x3vl3wajny6l52dlkncf2k58vztnlecjs0g5jqq
AGE-SECRET-KEY-15RKTPQCCLWM7EHQ8JEP0TQLUWJAECVP7332M3ZP0RL9R7JT7MZ6SY79V8Q`
