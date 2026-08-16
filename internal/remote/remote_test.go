package remote_test

import (
	"testing"

	. "github.com/npikall/gotpm/internal/remote"
	"github.com/stretchr/testify/assert"
)

func TestDefaultHTTPCloneURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"has http", "http://github.com/user/repo.git", "http://github.com/user/repo.git"},
		{"has no scheme", "github.com/user/repo.git", "https://github.com/user/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DefaultHTTPCloneURL(tt.got)
			assert.Equal(t, tt.want, got)
		})
	}
}
