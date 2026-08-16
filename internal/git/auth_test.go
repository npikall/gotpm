package git_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestHasHTTPTokenPrefersGotpmToken(t *testing.T) {
	for _, name := range git.TokenEnvVars() {
		t.Setenv(name, "")
	}
	assert.False(t, git.HasHTTPToken(), "no token set")

	t.Setenv("GITHUB_TOKEN", "  ")
	assert.False(t, git.HasHTTPToken(), "blank token does not count")

	t.Setenv("GITHUB_TOKEN", "from-github")
	assert.True(t, git.HasHTTPToken())
}

func TestTokenEnvVarsOrder(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"GOTPM_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"}, git.TokenEnvVars())
}

func TestTokenEnvVarsIsACopy(t *testing.T) {
	t.Parallel()
	git.TokenEnvVars()[0] = "MUTATED"
	assert.Equal(t, "GOTPM_TOKEN", git.TokenEnvVars()[0])
}
