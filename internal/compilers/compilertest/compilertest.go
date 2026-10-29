// Package compilertest serves fake Typst releases and a fake GitHub API, so
// compilers can be installed in tests without the network.
package compilertest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/npikall/gotpm/internal/compilers"
	"github.com/ulikunitz/xz"
)

// Modes of the files in a release archive.
const (
	execMode = 0o755
	fileMode = 0o644
)

// Server is a fake Typst repository publishing releases.
type Server struct {
	srv *httptest.Server

	mu       sync.Mutex
	releases []release
	archives map[string][]byte // by download path
	requests []string
}

type release struct {
	tag        string
	prerelease bool
	assets     []asset
}

type asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

// Option changes how a release is published.
type Option func(*release)

// Prerelease marks the release as a pre-release.
func Prerelease() Option {
	return func(r *release) { r.prerelease = true }
}

// WrongDigest publishes a digest that does not match the archive.
func WrongDigest() Option {
	return func(r *release) {
		for i := range r.assets {
			r.assets[i].Digest = "sha256:" + strings.Repeat("0", sha256.Size*2) //nolint: mnd
		}
	}
}

// NoDigest publishes the archive without a digest, as older releases are.
func NoDigest() Option {
	return func(r *release) {
		for i := range r.assets {
			r.assets[i].Digest = ""
		}
	}
}

// NoAssetForThisPlatform publishes the archive only for a platform other
// than this one, as for a platform Typst ships no compiler for.
func NoAssetForThisPlatform() Option {
	return func(r *release) {
		for i := range r.assets {
			r.assets[i].Name = "typst-elsewhere-unknown-none.tar.xz"
		}
	}
}

// New starts a fake repository, stopped when the test ends.
func New(t *testing.T) *Server {
	t.Helper()
	s := &Server{archives: make(map[string][]byte)}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

// Serve starts a fake repository and makes it the release source until the
// test ends. Tests using it must not run in parallel.
func Serve(t *testing.T) *Server {
	t.Helper()
	s := New(t)
	t.Cleanup(compilers.Use(s.Source()))
	return s
}

// Source is a GitHub release source reading from this server.
func (s *Server) Source() compilers.GitHub {
	return compilers.GitHub{APIURL: s.srv.URL + "/api"}
}

// BinaryContent is what the dummy typst binary of a version holds.
func BinaryContent(version string) string {
	return "typst " + version + "\n"
}

// Publish releases version with an archive for this platform holding a dummy
// typst binary, and the archive's digest.
func (s *Server) Publish(version string, opts ...Option) {
	name, archive := build(version)
	sum := sha256.Sum256(archive)
	download := "/download/v" + version + "/" + name
	r := release{tag: "v" + version, assets: []asset{{
		Name:   name,
		URL:    s.srv.URL + download,
		Digest: "sha256:" + hex.EncodeToString(sum[:]),
	}}}
	for _, opt := range opts {
		opt(&r)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.archives[download] = archive
	s.releases = append(s.releases, r)
}

// Unreachable takes the repository offline: every later request fails to
// connect.
func (s *Server) Unreachable() {
	s.srv.Close()
}

// Requests returns the path of every request received so far.
func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, r.URL.Path)

	switch {
	case r.URL.Path == "/api/releases":
		s.list(w)
	case strings.HasPrefix(r.URL.Path, "/api/releases/tags/"):
		s.tag(w, r, strings.TrimPrefix(r.URL.Path, "/api/releases/tags/"))
	case s.archives[r.URL.Path] != nil:
		_, _ = w.Write(s.archives[r.URL.Path])
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) list(w http.ResponseWriter) {
	out := make([]map[string]any, 0, len(s.releases))
	for _, r := range s.releases {
		out = append(out, r.json())
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) tag(w http.ResponseWriter, r *http.Request, tag string) {
	i := slices.IndexFunc(s.releases, func(r release) bool { return r.tag == tag })
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(s.releases[i].json())
}

func (r release) json() map[string]any {
	return map[string]any{"tag_name": r.tag, "prerelease": r.prerelease, "draft": false, "assets": r.assets}
}

// build packs the dummy binary of version the way Typst packs a release for
// this platform: inside a "typst-<target>" directory, zipped on windows and
// a tar.xz elsewhere.
func build(version string) (string, []byte) {
	name, err := compilers.AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		panic(err)
	}
	dir := strings.TrimSuffix(strings.TrimSuffix(name, ".zip"), ".tar.xz")
	content := []byte(BinaryContent(version))
	if runtime.GOOS == "windows" {
		return name, zipped(path.Join(dir, "typst.exe"), content)
	}
	return name, tarXz(path.Join(dir, "typst"), content)
}

func tarXz(file string, content []byte) []byte {
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	must(err)
	tw := tar.NewWriter(xw)
	must(tw.WriteHeader(&tar.Header{Name: path.Dir(file) + "/", Typeflag: tar.TypeDir, Mode: execMode}))
	license := []byte("license")
	must(tw.WriteHeader(&tar.Header{Name: path.Dir(file) + "/LICENSE", Typeflag: tar.TypeReg, Mode: fileMode, Size: int64(len(license))}))
	_, err = tw.Write(license)
	must(err)
	must(tw.WriteHeader(&tar.Header{Name: file, Typeflag: tar.TypeReg, Mode: execMode, Size: int64(len(content))}))
	_, err = tw.Write(content)
	must(err)
	must(tw.Close())
	must(xw.Close())
	return buf.Bytes()
}

func zipped(file string, content []byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(file)
	must(err)
	_, err = w.Write(content)
	must(err)
	must(zw.Close())
	return buf.Bytes()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
