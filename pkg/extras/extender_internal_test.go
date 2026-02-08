//nolint:dupl // there is some duplication in the test cases but it's not worth the abstraction
package extras

// cSpell: words lithammer sishserver holepunch citest uninode

import (
	"bytes"
	"testing"

	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml_utils "sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSplitPath(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	p := "toto.tata.!!yaml.toto.tata"
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	remainder, err := splitExtendedPath(path, &extensions)

	req.NoError(err)
	req.Len(remainder, 2, "Remainder path should be 2")
	req.Len(extensions, 1, "Should only have one extension")
	req.Equal("yaml", extensions[0].Encoding, "Extension should be yaml")
	req.Len(extensions[0].Path, 2, "Extension path len should be 2")
}

func TestRegexExtender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	text := dedent.Dedent(`
    PubkeyAcceptedKeyTypes +ssh-rsa
    Host sishserver
      HostName holepunch.in
      Port 2222
      BatchMode yes
      IdentityFile ~/.ssh_keys/id_rsa
      IdentitiesOnly yes
      LogLevel ERROR
      ServerAliveInterval 10
      ServerAliveCountMax 2
      RemoteCommand sni-proxy=true
      RemoteForward citest.holepunch.in:443 traefik.traefik.svc:443
    `)
	expected := dedent.Dedent(`
    PubkeyAcceptedKeyTypes +ssh-rsa
    Host sishserver
      HostName karmafun.dev
      Port 2222
      BatchMode yes
      IdentityFile ~/.ssh_keys/id_rsa
      IdentitiesOnly yes
      LogLevel ERROR
      ServerAliveInterval 10
      ServerAliveCountMax 2
      RemoteCommand sni-proxy=true
      RemoteForward citest.holepunch.in:443 traefik.traefik.svc:443
    `)
	path := &ExtendedSegment{
		Encoding: "regex",
		Path:     []string{`^\s+HostName\s+(\S+)\s*$`, `1`},
	}

	extender, err := path.Extender([]byte(text))
	req.NoError(err)
	req.NotNil(extender)

	req.NoError(extender.Set(path.Path, []byte("karmafun.dev")))

	out, err := extender.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(out), "Text should be modified")
}

func TestBase64Extender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	//nolint:lll // Base64 string, hard to break into multiple lines
	encoded := "UHVia2V5QWNjZXB0ZWRLZXlUeXBlcyArc3NoLXJzYQpIb3N0IHNpc2hzZXJ2ZXIKICBIb3N0TmFtZSBob2xlcHVuY2guaW4KICBQb3J0IDIyMjIKICBCYXRjaE1vZGUgeWVzCiAgSWRlbnRpdHlGaWxlIH4vLnNzaF9rZXlzL2lkX3JzYQogIElkZW50aXRpZXNPbmx5IHllcwogIExvZ0xldmVsIEVSUk9SCiAgU2VydmVyQWxpdmVJbnRlcnZhbCAxMAogIFNlcnZlckFsaXZlQ291bnRNYXggMgogIFJlbW90ZUNvbW1hbmQgc25pLXByb3h5PXRydWUKICBSZW1vdGVGb3J3YXJkIGNpdGVzdC5ob2xlcHVuY2guaW46NDQzIHRyYWVmaWsudHJhZWZpay5zdmM6NDQzCg=="
	decodedExpected := dedent.Dedent(`
	PubkeyAcceptedKeyTypes +ssh-rsa
	Host sishserver
	  HostName holepunch.in
	  Port 2222
	  BatchMode yes
	  IdentityFile ~/.ssh_keys/id_rsa
	  IdentitiesOnly yes
	  LogLevel ERROR
	  ServerAliveInterval 10
	  ServerAliveCountMax 2
	  RemoteCommand sni-proxy=true
	  RemoteForward citest.holepunch.in:443 traefik.traefik.svc:443
	`)[1:]

	//nolint:lll // Base64 string, hard to break into multiple lines
	modifiedEncoded := "UHVia2V5QWNjZXB0ZWRLZXlUeXBlcyArc3NoLXJzYQpIb3N0IHNpc2hzZXJ2ZXIKICBIb3N0TmFtZSBrYXJtYWZ1bi5kZXYKICBQb3J0IDIyMjIKICBCYXRjaE1vZGUgeWVzCiAgSWRlbnRpdHlGaWxlIH4vLnNzaF9rZXlzL2lkX3JzYQogIElkZW50aXRpZXNPbmx5IHllcwogIExvZ0xldmVsIEVSUk9SCiAgU2VydmVyQWxpdmVJbnRlcnZhbCAxMAogIFNlcnZlckFsaXZlQ291bnRNYXggMgogIFJlbW90ZUNvbW1hbmQgc25pLXByb3h5PXRydWUKICBSZW1vdGVGb3J3YXJkIGNpdGVzdC5ob2xlcHVuY2guaW46NDQzIHRyYWVmaWsudHJhZWZpay5zdmM6NDQzCg=="

	p := `!!base64.!!regex.\s+HostName\s+(\S+).1`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 2, "There should be 2 extensions")
	req.Equal("base64", extensions[0].Encoding, "The first extension should be base64")

	b64Ext := extensions[0]
	b64Extender, err := b64Ext.Extender([]byte(encoded))
	req.NoError(err)
	req.IsType(&base64Extender{}, b64Extender, "Should be a base64 extender")

	decoded, err := b64Extender.Get(b64Ext.Path)
	req.NoError(err)
	req.Equal(decodedExpected, string(decoded), "bad base64 decoding")

	regexExt := extensions[1]
	reExtender, err := regexExt.Extender(decoded)
	req.NoError(err)
	req.IsType(&regexExtender{}, reExtender, "Should be a regex extender")

	req.NoError(reExtender.Set(regexExt.Path, []byte("karmafun.dev")))
	modified, err := reExtender.GetPayload()
	req.NoError(err)
	req.NoError(b64Extender.Set(b64Ext.Path, modified))
	final, err := b64Extender.GetPayload()
	req.NoError(err)
	req.Equal(modifiedEncoded, string(final), "final base64 is bad")
}

func TestYamlExtender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	source := dedent.Dedent(`
    uninode: true
    common:
      targetRevision: main
    apps:
      enabled: true
    `)[1:]
	expected := dedent.Dedent(`
    uninode: true
    common:
      targetRevision: deploy/citest
    apps:
      enabled: true
    `)[1:]

	p := `!!yaml.common.targetRevision`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 1, "There should be 2 extensions")
	req.Equal("yaml", extensions[0].Encoding, "The first extension should be base64")

	yamlXP := extensions[0]
	yamlExt, err := yamlXP.Extender([]byte(source))
	req.NoError(err)
	value, err := yamlExt.Get(yamlXP.Path)
	req.NoError(err)
	req.Equal("main", string(value), "error fetching value")
	req.NoError(yamlExt.Set(yamlXP.Path, []byte("deploy/citest")))

	modified, err := yamlExt.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(modified), "final yaml")

	value, err = yamlExt.Get(yamlXP.Path)
	req.NoError(err)
	req.Equal("deploy/citest", string(value), "error fetching changed value")
}

func TestYamlExtenderWithSequence(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	source := dedent.Dedent(`
    - name: common.targetRevision
      value: main
    - name: common.repoURL
      value: https://github.com/antoinemartin/autocloud.git
    `)[1:]
	expected := dedent.Dedent(`
    - name: common.targetRevision
      value: deploy/citest
    - name: common.repoURL
      value: https://github.com/antoinemartin/autocloud.git
    `)[1:]

	p := `!!yaml.[name=common.targetRevision].value`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 1, "There should be 2 extensions")
	req.Equal("yaml", extensions[0].Encoding, "The first extension should be base64")

	yamlXP := extensions[0]
	yamlExt, err := yamlXP.Extender([]byte(source))
	req.NoError(err)
	req.NoError(yamlExt.Set(yamlXP.Path, []byte("deploy/citest")))

	modified, err := yamlExt.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(modified), "final yaml")
}

func TestYamlExtenderWithYaml(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	sources, err := (&kio.ByteReader{Reader: bytes.NewBufferString(`
common: |
  uninode: true
  common:
    targetRevision: main
  apps:
    enabled: true
`)}).Read()
	req.NoError(err)
	req.Len(sources, 1)
	source := sources[0]

	expected := dedent.Dedent(`
    common: |
      uninode: true
      common:
        targetRevision: deploy/citest
        repoURL: https://github.com/antoinemartin/autocloud.git
      apps:
        enabled: true
    `)[1:]

	replacements, err := (&kio.ByteReader{Reader: bytes.NewBufferString(`
common:
    targetRevision: deploy/citest
    repoURL: https://github.com/antoinemartin/autocloud.git
`)}).Read()
	req.NoError(err)
	req.Len(replacements, 1)
	replacement := replacements[0]

	p := `common.!!yaml.common`
	path := kyaml_utils.SmarterPathSplitter(p, ".")
	e, err := NewExtendedPath(path)
	req.NoError(err)
	req.Len(e.ResourcePath, 1, "no resource path")

	sourcePath := []string{"common"}

	target, err := source.Pipe(&yaml.PathGetter{Path: e.ResourcePath})
	req.NoError(err)

	value, err := replacement.Pipe(&yaml.PathGetter{Path: sourcePath})
	req.NoError(err)
	err = e.Apply(target, value)
	req.NoError(err)

	var b bytes.Buffer
	err = (&kio.ByteWriter{Writer: &b}).Write(sources)
	req.NoError(err)

	sString, err := source.String()
	req.NoError(err)
	req.Equal(expected, b.String(), sString, "replacement failed")
}

func TestJsonExtender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	source := `{
  "common": {
    "targetRevision": "main"
  },
  "uninode": true,
  "apps": {
    "enabled": true
  }
}`
	expected := `{
  "apps": {
    "enabled": true
  },
  "common": {
    "targetRevision": "deploy/citest"
  },
  "uninode": true
}
`

	p := `!!json.common.targetRevision`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 1, "There should be 2 extensions")
	req.Equal("json", extensions[0].Encoding, "The first extension should be json")

	jsonXP := extensions[0]
	jsonExt, err := jsonXP.Extender([]byte(source))
	req.NoError(err)
	value, err := jsonExt.Get(jsonXP.Path)
	req.NoError(err)
	req.Equal("main", string(value), "error fetching value")
	req.NoError(jsonExt.Set(jsonXP.Path, []byte("deploy/citest")))

	modified, err := jsonExt.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(modified), "final json")

	value, err = jsonExt.Get(jsonXP.Path)
	req.NoError(err)
	req.Equal("deploy/citest", string(value), "error fetching changed value")
}

func TestJsonArrayExtender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	source := `[
  {
    "name": "targetRevision",
    "value": "main"
  },
  {
    "name": "repoURL",
    "value": "https://github.com/example/example.git"
  }
]`
	expected := `[
  {
    "name": "targetRevision",
    "value": "deploy/citest"
  },
  {
    "name": "repoURL",
    "value": "https://github.com/example/example.git"
  }
]
`

	p := `!!json.[name=targetRevision].value`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 1, "There should be 2 extensions")
	req.Equal("json", extensions[0].Encoding, "The first extension should be json")

	jsonXP := extensions[0]
	jsonExt, err := jsonXP.Extender([]byte(source))
	req.NoError(err)
	value, err := jsonExt.Get(jsonXP.Path)
	req.NoError(err)
	req.Equal("main", string(value), "error fetching value")
	req.NoError(jsonExt.Set(jsonXP.Path, []byte("deploy/citest")))

	modified, err := jsonExt.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(modified), "final json")

	value, err = jsonExt.Get(jsonXP.Path)
	req.NoError(err)
	req.Equal("deploy/citest", string(value), "error fetching changed value")
}

func TestTomlExtender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	source := `
uninode = true
[common]
targetRevision = 'main'
[apps]
enabled = true
`
	expected := `uninode = true

[apps]
enabled = true

[common]
targetRevision = 'deploy/citest'
`

	p := `!!toml.common.targetRevision`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 1, "There should be 2 extensions")
	req.Equal("toml", extensions[0].Encoding, "The first extension should be toml")

	tomlXP := extensions[0]
	tomlExt, err := tomlXP.Extender([]byte(source))
	req.NoError(err)
	value, err := tomlExt.Get(tomlXP.Path)
	req.NoError(err)
	req.Equal("main", string(value), "error fetching value")
	req.NoError(tomlExt.Set(tomlXP.Path, []byte("deploy/citest")))

	modified, err := tomlExt.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(modified), "final toml")

	value, err = tomlExt.Get(tomlXP.Path)
	req.NoError(err)
	req.Equal("deploy/citest", string(value), "error fetching changed value")
}

func TestIniExtender(t *testing.T) {
	t.Parallel()
	req := require.New(t)
	source := `
uninode = true
[common]
targetRevision = main
[apps]
enabled = true
`
	expected := `uninode = true

[common]
targetRevision = deploy/citest

[apps]
enabled = true
`

	p := `!!ini.common.targetRevision`
	path := kyaml_utils.SmarterPathSplitter(p, ".")

	extensions := []*ExtendedSegment{}
	prefix, err := splitExtendedPath(path, &extensions)
	req.NoError(err)
	req.Empty(prefix, "There should be no prefix")
	req.Len(extensions, 1, "There should be 2 extensions")
	req.Equal("ini", extensions[0].Encoding, "The first extension should be ini")

	iniXP := extensions[0]
	iniExt, err := iniXP.Extender([]byte(source))
	req.NoError(err)
	value, err := iniExt.Get(iniXP.Path)
	req.NoError(err)
	req.Equal("main", string(value), "error fetching value")
	req.NoError(iniExt.Set(iniXP.Path, []byte("deploy/citest")))

	modified, err := iniExt.GetPayload()
	req.NoError(err)
	req.Equal(expected, string(modified), "final ini")

	value, err = iniExt.Get(iniXP.Path)
	req.NoError(err)
	req.Equal("deploy/citest", string(value), "error fetching changed value")
}
