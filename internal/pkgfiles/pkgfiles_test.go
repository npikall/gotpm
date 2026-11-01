package pkgfiles_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkgfiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, paths.EnsureDir(filepath.Dir(filepath.Join(dir, name))))
	require.NoError(t, paths.WriteFile(filepath.Join(dir, name), []byte(content)))
}

func collectNames(t *testing.T, src string) []string {
	t.Helper()
	jobs, err := pkgfiles.Collect(src, t.TempDir())
	require.NoError(t, err)
	names := make([]string, 0, len(jobs))
	for _, job := range jobs {
		rel, err := filepath.Rel(src, job.Src)
		require.NoError(t, err)
		names = append(names, filepath.ToSlash(rel))
	}
	sort.Strings(names)
	return names
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

func TestCollect_RespectsTypstIgnore(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "README.md", "")
	writeFile(t, src, ".typstignore", "README.md\n")

	assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
}

func TestCollect_RespectsGitIgnore(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "out.pdf", "")
	writeFile(t, src, ".gitignore", "*.pdf\n")

	assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
}

func TestCollect_GitIgnoreAndTypstIgnoreCombined(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "out.pdf", "")
	writeFile(t, src, "notes.md", "")
	writeFile(t, src, ".gitignore", "*.pdf\n")
	writeFile(t, src, ".typstignore", "notes.md\n")

	assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
}

func TestCollect_IgnoredDirectorySkipsContents(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join("build", "a.typ"), "")
	writeFile(t, src, filepath.Join("build", "nested", "b.typ"), "")
	writeFile(t, src, ".typstignore", "build\n")

	assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
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

func TestCopyFile_ReportsFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.typ")
	require.NoError(t, paths.WriteFile(src, []byte("x")))
	blocker := filepath.Join(dir, "file")
	require.NoError(t, paths.WriteFile(blocker, []byte("x")))
	existingDir := filepath.Join(dir, "existing")
	require.NoError(t, os.Mkdir(existingDir, 0o750))

	tests := map[string]struct{ src, dest, want string }{
		"missing source":       {filepath.Join(dir, "missing"), filepath.Join(dir, "out"), "opening source file"},
		"parent is a file":     {src, filepath.Join(blocker, "out"), "creating parent directories"},
		"destination is a dir": {src, existingDir, "creating destination file"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.ErrorContains(t, pkgfiles.CopyFile(tt.src, tt.dest), tt.want)
		})
	}
}

func TestCollect_ReportsAMissingSource(t *testing.T) {
	t.Parallel()

	_, err := pkgfiles.Collect(filepath.Join(t.TempDir(), "missing"), t.TempDir())

	require.ErrorContains(t, err, "walking")
}

func TestCollect_HonoursNestedGitIgnore(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, "out.pdf", "")
	writeFile(t, src, filepath.Join("sub", "out.pdf"), "")
	writeFile(t, src, filepath.Join("sub", "keep.typ"), "")
	writeFile(t, src, filepath.Join("sub", ".gitignore"), "*.pdf\n")

	assert.Equal(t, []string{"lib.typ", "out.pdf", "sub/keep.typ"}, collectNames(t, src))
}

func TestCollect_DeeperGitIgnoreOutranksShallower(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, filepath.Join("sub", "a.pdf"), "")
	writeFile(t, src, ".gitignore", "*.pdf\n")
	writeFile(t, src, filepath.Join("sub", ".gitignore"), "!a.pdf\n")

	assert.Equal(t, []string{"sub/a.pdf"}, collectNames(t, src))
}

func TestCollect_TypstIgnoreOutranksNestedGitIgnore(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join("assets", "thumbnail.png"), "")
	writeFile(t, src, filepath.Join("assets", ".gitignore"), "thumbnail.png\n")
	writeFile(t, src, ".typstignore", "!thumbnail.png\n")

	assert.Equal(t, []string{"assets/thumbnail.png", "lib.typ"}, collectNames(t, src))
}

func TestCollect_OnlyRootTypstIgnoreCounts(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, filepath.Join("sub", "notes.md"), "")
	writeFile(t, src, filepath.Join("sub", ".typstignore"), "notes.md\n")

	assert.Equal(t, []string{"sub/notes.md"}, collectNames(t, src))
}

func TestCollect_IgnoresMachineLocalGitExcludes(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join(".git", "info", "exclude"), "lib.typ\n")

	assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
}

func TestCollect_FailsOnAnUnreadableGitIgnore(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads files regardless of mode")
	}
	for name, ignoreFile := range map[string]string{
		"root":   ".gitignore",
		"nested": filepath.Join("sub", ".gitignore"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src := t.TempDir()
			writeFile(t, src, "lib.typ", "")
			writeFile(t, src, ignoreFile, "*.pdf\n")
			require.NoError(t, os.Chmod(filepath.Join(src, ignoreFile), 0))

			_, err := pkgfiles.Collect(src, t.TempDir())

			require.ErrorContains(t, err, ".gitignore")
		})
	}
}

const validManifest = "[package]\nname = \"p\"\nversion = \"0.1.0\"\nentrypoint = \"lib.typ\"\n"

func TestCollect_LeavesOutNamesNeverPartOfAPackage(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join(".git", "HEAD"), "")
	writeFile(t, src, filepath.Join(".jj", "repo", "store", "type"), "")
	writeFile(t, src, filepath.Join("nested", ".jj", "working_copy", "type"), "")
	writeFile(t, src, ".gitignore", "")
	writeFile(t, src, ".typstignore", "")
	writeFile(t, src, paths.ProvenanceFile, "")
	writeFile(t, src, filepath.Join("nested", ".git", "HEAD"), "")
	writeFile(t, src, filepath.Join("nested", "a.typ"), "")

	assert.Equal(t, []string{"lib.typ", "nested/a.typ"}, collectNames(t, src))
}

func TestCollect_ReadsIgnoreFileLines(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"blank lines":         "a.md\n\nb.md\n",
		"CRLF line endings":   "a.md\r\nb.md\r\n",
		"no trailing newline": "a.md\nb.md",
		"comments in between": "a.md\n# lib.typ\nb.md\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src := t.TempDir()
			writeFile(t, src, "lib.typ", "")
			writeFile(t, src, "a.md", "")
			writeFile(t, src, "b.md", "")
			writeFile(t, src, ".typstignore", content)

			assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
		})
	}
}

func TestCollect_DirectoryOnlyPatternSparesAFile(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "build", "")
	writeFile(t, src, filepath.Join("sub", "build", "x.typ"), "")
	writeFile(t, src, ".gitignore", "build/\n")

	assert.Equal(t, []string{"build"}, collectNames(t, src))
}

func TestCollect_NegationBelowDoubleStarDirectory(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, filepath.Join("docs", "drop.md"), "")
	writeFile(t, src, filepath.Join("docs", "keep.md"), "")
	writeFile(t, src, filepath.Join("docs", "deep", "drop.md"), "")
	writeFile(t, src, ".gitignore", "docs/**\n!docs/keep.md\n")

	assert.Equal(t, []string{"docs/keep.md"}, collectNames(t, src))
}

func TestCollect_DoubleStarDirectoryExcludesEverythingBelow(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join("docs", "a.md"), "")
	writeFile(t, src, filepath.Join("docs", "deep", "b.md"), "")
	writeFile(t, src, filepath.Join("other", "docs", "c.md"), "")
	writeFile(t, src, ".gitignore", "docs/**\n")

	assert.Equal(t, []string{"lib.typ", "other/docs/c.md"}, collectNames(t, src))
}

func TestCollect_HonoursManifestExclude(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "lib.typ", "")
	writeFile(t, src, filepath.Join("docs", "a.md"), "")
	writeFile(t, src, "typst.toml", validManifest+"exclude = [\"docs\"]\n")

	assert.Equal(t, []string{"lib.typ", "typst.toml"}, collectNames(t, src))
}

func TestCollect_IgnoresAManifestAboveThePackage(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	writeFile(t, parent, "typst.toml", validManifest+"exclude = [\"lib.typ\"]\n")
	src := filepath.Join(parent, "pkg")
	writeFile(t, src, "lib.typ", "")

	assert.Equal(t, []string{"lib.typ"}, collectNames(t, src))
}

func TestCollect_FailsOnAnInvalidManifest(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	writeFile(t, src, "typst.toml", "not toml [")

	_, err := pkgfiles.Collect(src, t.TempDir())

	require.ErrorContains(t, err, "exclude list")
}
