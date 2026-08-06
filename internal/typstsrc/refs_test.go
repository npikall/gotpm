package typstsrc_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/typstsrc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"single import", `#import "@preview/foo:0.1.0"`, []string{"@preview/foo:0.1.0"}},
		{
			"multiple imports",
			"#import \"@preview/foo:0.1.0\"\n#import \"@preview/bar:1.2.3\"",
			[]string{"@preview/foo:0.1.0", "@preview/bar:1.2.3"},
		},
		{"hyphenated name", `#import "@preview/my-pkg:0.1.0"`, []string{"@preview/my-pkg:0.1.0"}},
		{"digits in name", `#import "@preview/polylux2:0.1.0"`, []string{"@preview/polylux2:0.1.0"}},
		{"repeated import listed once", "@preview/foo:0.1.0 @preview/foo:0.1.0", []string{"@preview/foo:0.1.0"}},
		{"no imports", "#let x = 1", nil},
		{"other namespaces are left alone", `#import "@local/foo:0.1.0"`, nil},
		{"incomplete version", `#import "@preview/foo:0.1"`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			refs := typstsrc.FindRefs([]byte(tt.content))

			got := make([]string, 0, len(refs))
			for _, ref := range refs {
				got = append(got, ref.String())
			}
			if tt.want == nil {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindRefs_ParsesTheVersion(t *testing.T) {
	t.Parallel()
	refs := typstsrc.FindRefs([]byte(`#import "@preview/cetz:0.5.2"`))
	require.Len(t, refs, 1)

	assert.Equal(t, "preview", refs[0].Namespace)
	assert.Equal(t, "cetz", refs[0].Name)
	assert.Equal(t, 5, refs[0].Version.Minor)
	assert.Equal(t, 2, refs[0].Version.Patch)
}

func TestRewriteRefs(t *testing.T) {
	t.Parallel()
	t.Run("rewrites a listed package", func(t *testing.T) {
		t.Parallel()
		got := typstsrc.RewriteRefs([]byte(`#import "@preview/foo:0.1.0"`), map[string]string{"foo": "0.2.0"})
		assert.Equal(t, `#import "@preview/foo:0.2.0"`, string(got))
	})
	t.Run("rewrites several packages", func(t *testing.T) {
		t.Parallel()
		content := "#import \"@preview/foo:0.1.0\"\n#import \"@preview/bar:1.0.0\""
		got := typstsrc.RewriteRefs([]byte(content), map[string]string{"foo": "0.2.0", "bar": "2.0.0"})
		assert.Equal(t, "#import \"@preview/foo:0.2.0\"\n#import \"@preview/bar:2.0.0\"", string(got))
	})
	t.Run("leaves unlisted packages alone", func(t *testing.T) {
		t.Parallel()
		got := typstsrc.RewriteRefs([]byte(`#import "@preview/foo:0.1.0"`), map[string]string{"bar": "9.9.9"})
		assert.Equal(t, `#import "@preview/foo:0.1.0"`, string(got))
	})
	t.Run("rewrites every occurrence", func(t *testing.T) {
		t.Parallel()
		content := "@preview/foo:0.1.0 and again @preview/foo:0.1.0"
		got := typstsrc.RewriteRefs([]byte(content), map[string]string{"foo": "0.3.0"})
		assert.Equal(t, "@preview/foo:0.3.0 and again @preview/foo:0.3.0", string(got))
	})
	t.Run("a package name is not read as a pattern", func(t *testing.T) {
		t.Parallel()
		content := `#import "@preview/a.c:0.1.0"`
		got := typstsrc.RewriteRefs([]byte(content), map[string]string{"abc": "0.2.0"})
		assert.Equal(t, content, string(got), "abc must not match a.c")
	})
	t.Run("no updates leaves the content untouched", func(t *testing.T) {
		t.Parallel()
		content := `#import "@preview/foo:0.1.0"`
		got := typstsrc.RewriteRefs([]byte(content), nil)
		assert.Equal(t, content, string(got))
	})
}
