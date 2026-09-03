package pkgfiles_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkgfiles"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, paths.EnsureDir(filepath.Dir(filepath.Join(dir, name))))
	require.NoError(t, paths.WriteFile(filepath.Join(dir, name), []byte(content)))
}

func jobSrcBasenames(jobs []pkgfiles.Job) []string {
	names := make([]string, 0, len(jobs))
	for _, job := range jobs {
		names = append(names, filepath.Base(job.Src))
	}
	sort.Strings(names)
	return names
}

func matcherFromLines(t *testing.T, patterns ...string) *ignore.GitIgnore {
	t.Helper()
	return ignore.CompileIgnoreLines(patterns...)
}

func TestCopyTree(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	writeFile(t, src, "README.md", "readme")
	writeFile(t, src, filepath.Join("utils", "helper.typ"), "helper")

	require.NoError(t, pkgfiles.CopyTree(src, dst))
	assert.FileExists(t, filepath.Join(dst, "lib.typ"))
	assert.FileExists(t, filepath.Join(dst, "README.md"))
	assert.FileExists(t, filepath.Join(dst, "utils", "helper.typ"))
}

func TestCopyTree_HonoursIgnoreFiles(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	writeFile(t, src, "notes.md", "notes")
	writeFile(t, src, ".typstignore", "notes.md\n")

	require.NoError(t, pkgfiles.CopyTree(src, dst))
	assert.FileExists(t, filepath.Join(dst, "lib.typ"))
	assert.NoFileExists(t, filepath.Join(dst, "notes.md"))
	assert.NoFileExists(t, filepath.Join(dst, ".typstignore"))
}

func TestReadIgnoreLines(t *testing.T) {
	t.Parallel()
	t.Run("returns nil for non-existent file", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, pkgfiles.ReadIgnoreLines(filepath.Join(t.TempDir(), "nope")))
	})
	t.Run("returns trimmed non-empty lines", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, ".typstignore", "*.typ\nREADME.md\n")
		assert.Equal(t, []string{"*.typ", "README.md"},
			pkgfiles.ReadIgnoreLines(filepath.Join(dir, ".typstignore")))
	})
	t.Run("skips blank lines", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, ".typstignore", "*.typ\n\nREADME.md\n")
		assert.Equal(t, []string{"*.typ", "README.md"},
			pkgfiles.ReadIgnoreLines(filepath.Join(dir, ".typstignore")))
	})
	t.Run("strips windows-style CRLF line endings", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, ".typstignore", "*.typ\r\nREADME.md\r\n")
		assert.Equal(t, []string{"*.typ", "README.md"},
			pkgfiles.ReadIgnoreLines(filepath.Join(dir, ".typstignore")))
	})
}

func TestShouldIgnore(t *testing.T) {
	t.Parallel()
	t.Run("root . is never ignored", func(t *testing.T) {
		t.Parallel()
		assert.False(t, pkgfiles.ShouldIgnore(".", nil))
	})
	t.Run("hardcoded filenames are ignored", func(t *testing.T) {
		t.Parallel()
		assert.True(t, pkgfiles.ShouldIgnore(".git", nil))
		assert.True(t, pkgfiles.ShouldIgnore(".gitignore", nil))
		assert.True(t, pkgfiles.ShouldIgnore(".typstignore", nil))
		assert.True(t, pkgfiles.ShouldIgnore(paths.ProvenanceFile, nil))
		assert.True(t, pkgfiles.ShouldIgnore(filepath.Join("nested", ".git"), nil))
	})
	t.Run("nil matcher does not ignore unknown files", func(t *testing.T) {
		t.Parallel()
		assert.False(t, pkgfiles.ShouldIgnore("lib.typ", nil))
	})
	t.Run("matcher-matched path is ignored", func(t *testing.T) {
		t.Parallel()
		matcher := matcherFromLines(t, "*.md")
		assert.True(t, pkgfiles.ShouldIgnore("README.md", matcher))
		assert.False(t, pkgfiles.ShouldIgnore("lib.typ", matcher))
	})
}

func TestCollect_RespectsTypstIgnore(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "README.md", "")
	writeFile(t, src, ".typstignore", "README.md\n")

	jobs, err := pkgfiles.Collect(src, t.TempDir(), pkgfiles.Matcher(src))
	require.NoError(t, err)
	assert.Equal(t, []string{"lib.typ"}, jobSrcBasenames(jobs))
}

func TestCollect_RespectsGitIgnore(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "out.pdf", "")
	writeFile(t, src, ".gitignore", "*.pdf\n")

	jobs, err := pkgfiles.Collect(src, t.TempDir(), pkgfiles.Matcher(src))
	require.NoError(t, err)
	assert.Equal(t, []string{"lib.typ"}, jobSrcBasenames(jobs))
}

func TestCollect_GitIgnoreAndTypstIgnoreCombined(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "out.pdf", "")
	writeFile(t, src, "notes.md", "")
	writeFile(t, src, ".gitignore", "*.pdf\n")
	writeFile(t, src, ".typstignore", "notes.md\n")

	jobs, err := pkgfiles.Collect(src, t.TempDir(), pkgfiles.Matcher(src))
	require.NoError(t, err)
	assert.Equal(t, []string{"lib.typ"}, jobSrcBasenames(jobs))
}

func TestCollect_IgnoredDirectorySkipsContents(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join("build", "a.typ"), "")
	writeFile(t, src, filepath.Join("build", "nested", "b.typ"), "")
	writeFile(t, src, ".typstignore", "build\n")

	jobs, err := pkgfiles.Collect(src, t.TempDir(), pkgfiles.Matcher(src))
	require.NoError(t, err)
	assert.Equal(t, []string{"lib.typ"}, jobSrcBasenames(jobs))
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	t.Run("copies content to dest", func(t *testing.T) {
		t.Parallel()
		src := filepath.Join(t.TempDir(), "src.typ")
		dst := filepath.Join(t.TempDir(), "dst.typ")
		require.NoError(t, paths.WriteFile(src, []byte("hello")))

		require.NoError(t, pkgfiles.CopyFile(src, dst))
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(got))
	})
	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()
		src := filepath.Join(t.TempDir(), "src.typ")
		dst := filepath.Join(t.TempDir(), "a", "b", "c", "dst.typ")
		require.NoError(t, paths.WriteFile(src, []byte("x")))

		require.NoError(t, pkgfiles.CopyFile(src, dst))
		assert.FileExists(t, dst)
	})
	t.Run("preserves file mode", func(t *testing.T) {
		t.Parallel()
		src := filepath.Join(t.TempDir(), "exec.typ")
		dst := filepath.Join(t.TempDir(), "exec-dst.typ")
		require.NoError(t, os.WriteFile(src, []byte("x"), paths.DirPerm))

		require.NoError(t, pkgfiles.CopyFile(src, dst))
		info, err := os.Stat(dst)
		require.NoError(t, err)
		assert.Equal(t, paths.DirPerm, info.Mode().Perm())
	})
	t.Run("missing source returns error", func(t *testing.T) {
		t.Parallel()
		err := pkgfiles.CopyFile("/does/not/exist.typ", filepath.Join(t.TempDir(), "dst.typ"))
		assert.ErrorContains(t, err, "opening source file")
	})
}

func TestRun(t *testing.T) {
	t.Parallel()
	t.Run("copies all jobs successfully", func(t *testing.T) {
		t.Parallel()
		srcDir := t.TempDir()
		dstDir := t.TempDir()
		writeFile(t, srcDir, "a.typ", "a")
		writeFile(t, srcDir, "b.typ", "b")

		jobs := []pkgfiles.Job{
			{Src: filepath.Join(srcDir, "a.typ"), Dst: filepath.Join(dstDir, "a.typ")},
			{Src: filepath.Join(srcDir, "b.typ"), Dst: filepath.Join(dstDir, "b.typ")},
		}

		require.NoError(t, pkgfiles.Run(jobs))
		assert.FileExists(t, filepath.Join(dstDir, "a.typ"))
		assert.FileExists(t, filepath.Join(dstDir, "b.typ"))
	})
	t.Run("aggregates errors from failed jobs", func(t *testing.T) {
		t.Parallel()
		jobs := []pkgfiles.Job{
			{Src: "/does/not/exist/a.typ", Dst: filepath.Join(t.TempDir(), "a.typ")},
			{Src: "/does/not/exist/b.typ", Dst: filepath.Join(t.TempDir(), "b.typ")},
		}
		assert.Error(t, pkgfiles.Run(jobs))
	})
	t.Run("empty job list returns no error", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, pkgfiles.Run(nil))
	})
}
