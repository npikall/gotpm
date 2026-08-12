package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage"
)

// File is one file to be written into the fork, named by its slash-separated
// path relative to the directory being published.
type File struct {
	// Path is relative to the directory the files are written into.
	Path string
	// Source is the file on disk to read the content from.
	Source string
	// Executable records whether the file keeps its executable bit.
	Executable bool
}

// writeDir builds the tree for a directory of files, nesting them by path.
func writeDir(st storage.Storer, files []File) (plumbing.Hash, error) {
	// Group the files by their first path segment, so each subdirectory can be
	// built by the same routine one level down.
	var here []object.TreeEntry
	subdirs := map[string][]File{}
	var order []string

	for _, file := range files {
		dir, rest, nested := strings.Cut(file.Path, "/")
		if !nested {
			hash, err := writeBlob(st, file.Source)
			if err != nil {
				return plumbing.ZeroHash, err
			}
			here = append(here, object.TreeEntry{
				Name: dir, Mode: fileMode(file), Hash: hash,
			})
			continue
		}
		if _, ok := subdirs[dir]; !ok {
			order = append(order, dir)
		}
		subdirs[dir] = append(subdirs[dir], File{
			Path: rest, Source: file.Source, Executable: file.Executable,
		})
	}

	for _, dir := range order {
		hash, err := writeDir(st, subdirs[dir])
		if err != nil {
			return plumbing.ZeroHash, err
		}
		here = append(here, object.TreeEntry{Name: dir, Mode: filemode.Dir, Hash: hash})
	}

	return writeTree(st, here)
}

func fileMode(file File) filemode.FileMode {
	if file.Executable {
		return filemode.Executable
	}
	return filemode.Regular
}

// splice rewrites the chain of trees from root down to dir so that dir holds
// leaf, and returns the new root tree. Directories off the path are carried
// over by hash and never read, which is what lets this work on a clone holding
// only the trees along the path.
func splice(
	st storage.Storer, root plumbing.Hash, dir []string, leaf plumbing.Hash,
) (plumbing.Hash, error) {
	tree, err := object.GetTree(st, root)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("reading tree %s: %w", root, err)
	}

	child := leaf
	if len(dir) > 1 {
		var sub plumbing.Hash
		if entry, err := tree.FindEntry(dir[0]); err == nil && entry.Mode == filemode.Dir {
			sub = entry.Hash
		}
		if sub.IsZero() {
			child, err = newDirs(st, dir[1:], leaf)
		} else {
			child, err = splice(st, sub, dir[1:], leaf)
		}
		if err != nil {
			return plumbing.ZeroHash, err
		}
	}

	return writeTree(st, replaceDir(tree.Entries, dir[0], child))
}

// newDirs builds a fresh chain of directories ending in leaf, for the part of a
// path that does not exist in the fork yet. The tree it returns holds dir[0],
// which points at a tree holding dir[1], and so on down to leaf - so a package
// published for the first time still lands under its version directory rather
// than replacing the package directory itself.
func newDirs(st storage.Storer, dir []string, leaf plumbing.Hash) (plumbing.Hash, error) {
	hash := leaf
	for _, name := range slices.Backward(dir) {
		var err error
		hash, err = writeTree(st, []object.TreeEntry{
			{Name: name, Mode: filemode.Dir, Hash: hash},
		})
		if err != nil {
			return plumbing.ZeroHash, err
		}
	}
	return hash, nil
}

// replaceDir returns entries with name pointing at hash, appending it when the
// directory is new. The result is unsorted; writeTree sorts it.
func replaceDir(entries []object.TreeEntry, name string, hash plumbing.Hash) []object.TreeEntry {
	out := make([]object.TreeEntry, 0, len(entries)+1)
	replaced := false
	for _, entry := range entries {
		if entry.Name == name {
			entry.Mode, entry.Hash = filemode.Dir, hash
			replaced = true
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, object.TreeEntry{Name: name, Mode: filemode.Dir, Hash: hash})
	}
	return out
}

// writeTree stores a tree, sorting its entries first.
//
// Tree.Encode rejects unsorted entries but go-git exposes no helper for the
// ordering it demands, which is not a plain sort by name: git compares
// directories as though their names ended in a slash, so that "foo.typ" sorts
// before the directory "foo".
func writeTree(st storage.Storer, entries []object.TreeEntry) (plumbing.Hash, error) {
	sortName := func(entry object.TreeEntry) string {
		if entry.Mode == filemode.Dir {
			return entry.Name + "/"
		}
		return entry.Name
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return sortName(entries[i]) < sortName(entries[j])
	})

	obj := st.NewEncodedObject()
	if err := (&object.Tree{Entries: entries}).Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encoding tree: %w", err)
	}
	hash, err := st.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing tree: %w", err)
	}
	return hash, nil
}

// writeBlob stores the content of a file on disk as a blob.
func writeBlob(st storage.Storer, source string) (plumbing.Hash, error) {
	file, err := os.Open(source) //nolint: gosec // the caller collected this path
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("opening %q: %w", source, err)
	}
	defer file.Close() //nolint: errcheck

	obj := st.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	writer, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("writing blob for %q: %w", source, err)
	}
	if _, err := io.Copy(writer, file); err != nil {
		return plumbing.ZeroHash, errors.Join(fmt.Errorf("reading %q: %w", source, err), writer.Close())
	}
	if err := writer.Close(); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("writing blob for %q: %w", source, err)
	}

	hash, err := st.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing blob for %q: %w", source, err)
	}
	return hash, nil
}

// splitPath breaks a slash-separated repository path into its segments.
func splitPath(dir string) []string {
	return strings.Split(path.Clean(dir), "/")
}
