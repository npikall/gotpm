// Package fonttest serves a fake Google Fonts repository and GitHub API, so
// font commands can be tested without the network.
package fonttest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/npikall/gotpm/internal/fonts"
)

// Server is a fake Google Fonts repository with a commit history per family.
type Server struct {
	srv *httptest.Server

	mu       sync.Mutex
	commits  map[string][]commit // by "<license>/<key>", oldest first
	requests []Request
	serial   int
}

// Request is one request the server received.
type Request struct {
	Path          string
	Authorization string
}

type commit struct {
	sha   string
	files map[string]string
}

// New starts a fake repository, stopped when the test ends.
func New(t *testing.T) *Server {
	t.Helper()
	s := &Server{commits: make(map[string][]commit)}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

// Serve starts a fake repository and makes it the Google font source until the
// test ends. Tests using it must not run in parallel.
func Serve(t *testing.T) *Server {
	t.Helper()
	s := New(t)
	t.Cleanup(fonts.Use(s.Source()))
	return s
}

// Source is a Google font source reading from this server.
func (s *Server) Source() fonts.Google {
	return fonts.Google{RawURL: s.srv.URL + "/raw", APIURL: s.srv.URL + "/api"}
}

// Publish commits a family's font files, plus a METADATA.pb listing them, and
// returns the commit. The newest commit is what the family resolves to.
func (s *Server) Publish(license, key string, files ...string) string {
	contents := make(map[string]string, len(files)+1)
	for _, file := range files {
		contents[file] = file + " content"
	}
	return s.PublishContents(license, key, contents)
}

// PublishContents is Publish with the content of every file spelled out.
func (s *Server) PublishContents(license, key string, files map[string]string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serial++
	sha := fmt.Sprintf("%040x", s.serial)
	contents := make(map[string]string, len(files)+1)
	var metadata strings.Builder
	fmt.Fprintf(&metadata, "name: %q\n", key)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		contents[name] = files[name]
		fmt.Fprintf(&metadata, "fonts {\n  filename: %q\n}\n", name)
	}
	contents["METADATA.pb"] = metadata.String()
	path := license + "/" + key
	s.commits[path] = append(s.commits[path], commit{sha: sha, files: contents})
	return sha
}

// Requests returns every request received so far.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// Downloads counts the font files served, leaving out metadata.
func (s *Server) Downloads() int {
	count := 0
	for _, r := range s.Requests() {
		if strings.HasPrefix(r.Path, "/raw/") && !strings.HasSuffix(r.Path, "/METADATA.pb") {
			count++
		}
	}
	return count
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, Request{Path: r.URL.Path, Authorization: r.Header.Get("Authorization")})

	routes := []struct {
		prefix string
		serve  func(w http.ResponseWriter, r *http.Request, rest string)
	}{
		{"/raw/", s.raw},
		{"/api/commits", s.latestCommit},
		{"/api/git/trees/", s.tree},
	}
	for _, route := range routes {
		if rest, ok := strings.CutPrefix(r.URL.Path, route.prefix); ok {
			route.serve(w, r, rest)
			return
		}
	}
	http.NotFound(w, r)
}

// raw serves "<ref>/<license>/<key>/<file>", where ref is a commit or "main".
func (s *Server) raw(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.SplitN(rest, "/", 4) //nolint: mnd
	if len(parts) != 4 {                  //nolint: mnd
		http.NotFound(w, r)
		return
	}
	body, ok := s.find(parts[0], parts[1]+"/"+parts[2]).files[parts[3]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(body))
}

// find returns the commit of a family directory a ref names, the newest for
// "main", or an empty commit.
func (s *Server) find(ref, path string) commit {
	history := s.commits[path]
	if ref == "main" {
		return newest(history)
	}
	i := slices.IndexFunc(history, func(c commit) bool { return c.sha == ref })
	if i < 0 {
		return commit{}
	}
	return history[i]
}

func newest(history []commit) commit {
	if len(history) == 0 {
		return commit{}
	}
	return history[len(history)-1]
}

func (s *Server) latestCommit(w http.ResponseWriter, r *http.Request, _ string) {
	type ghCommit struct {
		SHA string `json:"sha"`
	}
	out := []ghCommit{}
	if c := s.find("main", r.URL.Query().Get("path")); c.sha != "" {
		out = append(out, ghCommit{SHA: c.sha})
	}
	_ = json.NewEncoder(w).Encode(out)
}

// treeType marks a directory in a GitHub tree.
const treeType = "tree"

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

// tree answers the root tree with the license directories, and "tree-<license>"
// with the families published under it.
func (s *Server) tree(w http.ResponseWriter, r *http.Request, sha string) {
	entries := []treeEntry{{Path: "README.md", Type: "blob", SHA: "readme"}}
	if sha == "main" {
		entries = append(entries, rootTree()...)
	} else if license, ok := strings.CutPrefix(sha, "tree-"); ok {
		entries = append(entries, s.licenseTree(license)...)
	} else {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"tree": entries, "truncated": false})
}

func rootTree() []treeEntry {
	licenses := []string{"apache", "ofl", "ufl"}
	entries := make([]treeEntry, 0, len(licenses))
	for _, license := range licenses {
		entries = append(entries, treeEntry{Path: license, Type: treeType, SHA: "tree-" + license})
	}
	return entries
}

func (s *Server) licenseTree(license string) []treeEntry {
	var entries []treeEntry
	for path := range s.commits {
		if l, key, _ := strings.Cut(path, "/"); l == license {
			entries = append(entries, treeEntry{Path: key, Type: treeType, SHA: "family"})
		}
	}
	return entries
}
