package cmd_test

import (
	"testing"

	"github.com/npikall/gotpm/cmd"
	"github.com/stretchr/testify/assert"
)

func TestUpdateFileContent(t *testing.T) {
	got := []byte(`#import "@preview/foo:0.1.0"`)
	versions := map[string]string{"foo": "0.2.0"}

	cmd.UpdateFileContent(&got, versions)

	want := []byte(`#import "@preview/foo:0.2.0"`)
	assert.Equal(t, string(want), string(got))
}
