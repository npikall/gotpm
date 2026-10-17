package testrepo

import (
	"net/http"
	"net/http/cgi" //nolint: gosec // test fixture; httpoxy only affects Go < 1.6.3
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/require"
)

const (
	privateOwner    = "owner"
	privateUser     = "reader"
	privatePassword = "secret"
)

// Private serves the package's repository over http behind basic auth, the way
// a host answers for a private repository: 401 to anyone without credentials.
// Its URL changes to the served one, so call it before anything locks it.
func (p *Package) Private() *Package {
	t := p.t
	t.Helper()
	git, err := exec.LookPath("git")
	require.NoError(t, err)

	backend := &cgi.Handler{
		Path: git,
		Args: []string{"http-backend"},
		Root: "/" + privateOwner,
		Env: []string{
			"GIT_PROJECT_ROOT=" + filepath.Dir(p.dir),
			"GIT_HTTP_EXPORT_ALL=1",
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL=" + os.DevNull,
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != privateUser || password != privatePassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="private"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	p.served = server.URL + "/" + privateOwner + "/" + filepath.Base(p.dir)
	return p
}

// GrantAccess gives git, and so gotpm, the credentials every private package
// accepts, through a credential helper in an isolated global git config. Call
// it after Isolate, which resets that config.
func GrantAccess(t *testing.T) {
	t.Helper()
	config := filepath.Join(t.TempDir(), "gitconfig")
	helper := `!f() { test "$1" = get && printf 'username=` + privateUser + `\npassword=` + privatePassword + `\n'; }; f`
	content := "[credential]\n\thelper = \"" + helper + "\"\n"
	require.NoError(t, paths.WriteFile(config, []byte(content)))
	t.Setenv("GIT_CONFIG_GLOBAL", config)
}
