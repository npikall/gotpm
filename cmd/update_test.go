package cmd_test

import (
	"testing"

	"github.com/npikall/gotpm/cmd"
	"github.com/stretchr/testify/assert"
)

func TestUpdateFileContent(t *testing.T) {
	got := []byte(`#import "@preview/foo:0.1.0"`)
	versions := map[string]cmd.Result{"foo": cmd.Result{Name: "foo", Latest: "0.2.0", Current: "0.1.0"}}

	cmd.UpdateFileContent(&got, versions)

	want := []byte(`#import "@preview/foo:0.2.0"`)
	assert.Equal(t, string(want), string(got))
}
