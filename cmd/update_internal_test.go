package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePackageRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"@preview/foo:0.1.0", "foo", "0.1.0"},
		{"@preview/my-pkg:2.3.4", "my-pkg", "2.3.4"},
		{"@preview/bar:0.0.1", "bar", "0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			name, version := ParsePackageRef([]byte(tt.input))
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestExtractImportStatements(t *testing.T) {
	t.Parallel()
	t.Run("finds single import", func(t *testing.T) {
		t.Parallel()
		content := []byte(`#import "@preview/foo:0.1.0": *`)
		got := ExtractImportStatements(content)
		require.Len(t, got, 1)
		assert.Equal(t, "@preview/foo:0.1.0", string(got[0]))
	})
	t.Run("finds multiple imports", func(t *testing.T) {
		t.Parallel()
		content := []byte(`#import "@preview/foo:0.1.0": *
#import "@preview/bar:2.3.4": *`)
		got := ExtractImportStatements(content)
		assert.Len(t, got, 2)
	})
	t.Run("no imports returns empty slice", func(t *testing.T) {
		t.Parallel()
		content := []byte(`#let x = 1`)
		got := ExtractImportStatements(content)
		assert.Empty(t, got)
	})
	t.Run("non-preview imports are not matched", func(t *testing.T) {
		t.Parallel()
		content := []byte(`#import "@local/foo:0.1.0": *`)
		got := ExtractImportStatements(content)
		assert.Empty(t, got)
	})
	t.Run("deduplication not done — duplicate imports both returned", func(t *testing.T) {
		t.Parallel()
		content := []byte(`#import "@preview/foo:0.1.0": *
#import "@preview/foo:0.1.0": *`)
		got := ExtractImportStatements(content)
		assert.Len(t, got, 2)
	})
}

func TestUpdateFileContent_MultiplePackages(t *testing.T) {
	t.Parallel()
	content := []byte(`#import "@preview/foo:0.1.0": *
#import "@preview/bar:1.0.0": *`)
	versions := map[string]Result{
		"foo": {Name: "foo", Latest: "0.2.0", Current: "0.1.0"},
		"bar": {Name: "bar", Latest: "1.1.0", Current: "1.0.0"},
	}
	UpdateFileContent(&content, versions)
	assert.Contains(t, string(content), "@preview/foo:0.2.0")
	assert.Contains(t, string(content), "@preview/bar:1.1.0")
}

func TestUpdateFileContent_NoChange_WhenNotInMap(t *testing.T) {
	t.Parallel()
	content := []byte(`#import "@preview/foo:0.1.0": *`)
	UpdateFileContent(&content, map[string]Result{})
	assert.Equal(t, `#import "@preview/foo:0.1.0": *`, string(content))
}

func TestUpdateFileContent_ReplacesAllOccurrences(t *testing.T) {
	t.Parallel()
	content := []byte(`@preview/foo:0.1.0 and @preview/foo:0.1.0`)
	versions := map[string]Result{
		"foo": {Name: "foo", Latest: "0.2.0", Current: "0.1.0"},
	}
	UpdateFileContent(&content, versions)
	assert.Equal(t, `@preview/foo:0.2.0 and @preview/foo:0.2.0`, string(content))
}

func TestWriteOutputContent_ToOutputPath(t *testing.T) {
	t.Parallel()
	dst := filepath.Join(t.TempDir(), "out.typ")
	require.NoError(t, WriteOutputContent([]byte("hello"), "input.typ", dst))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestWriteOutputContent_ToInputFilePath(t *testing.T) {
	t.Parallel()
	dst := filepath.Join(t.TempDir(), "in.typ")
	require.NoError(t, WriteOutputContent([]byte("world"), dst, ""))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "world", string(got))
}

func TestLookupVersion_IndexHit(t *testing.T) {
	t.Parallel()
	index := map[string]string{"my-pkg": "3.0.0"}
	version, source, err := LookupVersion(t.Context(), index, "my-pkg")
	require.NoError(t, err)
	assert.Equal(t, "3.0.0", version)
	assert.Equal(t, "index", source)
}
