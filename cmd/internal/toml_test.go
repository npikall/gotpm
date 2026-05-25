package internal

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseTOML = `
[package]
name = "old-name"
version = "0.1.0"
entrypoint = "old.typ"
`

func TestConfigurableUpdateToml(t *testing.T) {
	newMeta := PackageMeta{Name: "new-name", Version: "1.2.3", Entrypoint: "lib.typ"}

	t.Run("updates all package fields", func(t *testing.T) {
		var buf bytes.Buffer
		err := ConfigurableUpdateToml(&buf, newMeta, []byte(baseTOML), toml.Unmarshal, false)
		require.NoError(t, err)

		var got map[string]any
		require.NoError(t, toml.Unmarshal(buf.Bytes(), &got))
		pkg := got["package"].(map[string]any)
		assert.Equal(t, "new-name", pkg["name"])
		assert.Equal(t, "1.2.3", pkg["version"])
		assert.Equal(t, "lib.typ", pkg["entrypoint"])
	})

	t.Run("missing package section returns ErrInvalidManifest", func(t *testing.T) {
		var buf bytes.Buffer
		err := ConfigurableUpdateToml(&buf, newMeta, []byte("[other]\nkey = \"val\"\n"), toml.Unmarshal, false)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})

	t.Run("custom unmarshal error propagates", func(t *testing.T) {
		sentinel := errors.New("unmarshal failed")
		failUnmarshal := func(_ []byte, _ any) error { return sentinel }
		var buf bytes.Buffer
		err := ConfigurableUpdateToml(&buf, newMeta, []byte(baseTOML), failUnmarshal, false)
		assert.ErrorIs(t, err, sentinel)
	})
}

func TestUpdateTOML(t *testing.T) {
	meta := PackageMeta{Name: "pkg", Version: "2.0.0", Entrypoint: "main.typ"}
	var buf bytes.Buffer
	err := UpdateTOML(&buf, meta, []byte(baseTOML), false)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "2.0.0", "output must contain new version")
	assert.Contains(t, out, "pkg", "output must contain new name")
	assert.Contains(t, out, "main.typ", "output must contain new entrypoint")
}
