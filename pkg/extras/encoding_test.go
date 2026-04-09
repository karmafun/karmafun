package extras_test

// cSpell: words karmafun bcrypt

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/karmafun/karmafun/pkg/extras"
)

func TestEncodeBase64(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.EncodeBase64("hello world")
	req.NoError(err)
	decoded, err := base64.StdEncoding.DecodeString(result)
	req.NoError(err)
	req.Equal("hello world", string(decoded))
}

func TestEncodeBase64_Empty(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.EncodeBase64("")
	req.NoError(err)
	req.Equal(base64.StdEncoding.EncodeToString([]byte("")), result)
}

func TestEncodeBcrypt(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.EncodeBcrypt("mypassword")
	req.NoError(err)
	req.NotEmpty(result)
	// Verify the hash is valid bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(result), []byte("mypassword"))
	req.NoError(err)
}

func TestEncodeHex(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.EncodeHex("hello")
	req.NoError(err)
	req.Equal("68656c6c6f", result)
}

func TestEncodeHex_Empty(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.EncodeHex("")
	req.NoError(err)
	req.Equal("", result)
}

func TestGetEncodedValue_Base64(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.GetEncodedValue("hello", "base64")
	req.NoError(err)
	req.Equal(base64.StdEncoding.EncodeToString([]byte("hello")), result)
}

func TestGetEncodedValue_Hex(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.GetEncodedValue("hello", "hex")
	req.NoError(err)
	req.Equal("68656c6c6f", result)
}

func TestGetEncodedValue_Bcrypt(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	result, err := extras.GetEncodedValue("mypassword", "bcrypt")
	req.NoError(err)
	req.NotEmpty(result)
	err = bcrypt.CompareHashAndPassword([]byte(result), []byte("mypassword"))
	req.NoError(err)
}

func TestGetEncodedValue_Unknown(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	_, err := extras.GetEncodedValue("hello", "unknown-encoding")
	req.Error(err)
	req.Contains(err.Error(), "unknown")
}

func TestGetEncodedValue_CaseInsensitive(t *testing.T) {
	t.Parallel()
	req := require.New(t)

	// Should work case-insensitively
	result, err := extras.GetEncodedValue("hello", "BASE64")
	req.NoError(err)
	req.Equal(base64.StdEncoding.EncodeToString([]byte("hello")), result)
}
