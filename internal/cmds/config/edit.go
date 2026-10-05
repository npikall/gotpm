package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	cfgfile "github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/ui"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// errEditorFailed marks an editor that exited with an error, which is taken
// to mean the user abandoned the edit.
var errEditorFailed = errors.New("editor failed")

// Edit opens the config file in the user's editor. The editor works on a
// draft, which replaces the config file only once it parses, so a broken edit
// never reaches the file publish reads. A config file that does not exist yet
// starts out as cfgfile.Template.
func Edit(ctx context.Context) error {
	draft, path, err := createDraft()
	if err != nil {
		return err
	}
	defer os.Remove(draft) //nolint: errcheck // gone already once it is saved

	for err := editOnce(ctx, draft); err != nil; err = editOnce(ctx, draft) {
		if !retry(err, ui.IsTerminal(), ui.Confirm) {
			return err
		}
	}
	return save(draft, path)
}

// editOnce lets the user edit draft and reports whether it is a valid config.
func editOnce(ctx context.Context, draft string) error {
	if err := runEditor(ctx, draft); err != nil {
		return fmt.Errorf("%w: %w", errEditorFailed, err)
	}
	return validate(draft)
}

// retry reports whether the user wants to fix an invalid draft, which is only
// asked when somebody is there to answer and the editor did not fail.
func retry(err error, interactive bool, ask func(string) (bool, error)) bool {
	if !interactive || errors.Is(err, errEditorFailed) {
		return false
	}
	ui.Error(err)
	again, askErr := ask("Edit again?")
	return again && askErr == nil
}

func save(draft, path string) error {
	if err := os.Rename(draft, path); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	ui.Infof("saved %s", path)
	return nil
}

// createDraft copies the config file, or the template if there is none, to a
// new file beside it. It returns the draft's path and the config file's.
func createDraft() (string, string, error) {
	path, err := cfgfile.Path()
	if err != nil {
		return "", "", err
	}
	content, err := readOrTemplate(path)
	if err != nil {
		return "", "", err
	}
	draft, err := writeDraft(filepath.Dir(path), content)
	return draft, path, err
}

func readOrTemplate(path string) ([]byte, error) {
	content, err := os.ReadFile(path) //nolint: gosec
	if errors.Is(err, fs.ErrNotExist) {
		return []byte(cfgfile.Template), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	return content, nil
}

// writeDraft writes content to a new temporary file in dir.
func writeDraft(dir string, content []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint: gosec, mnd
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	draft, err := os.CreateTemp(dir, ".config-*.toml")
	if err != nil {
		return "", fmt.Errorf("creating draft: %w", err)
	}
	defer draft.Close() //nolint: errcheck
	if _, err := draft.Write(content); err != nil {
		return "", fmt.Errorf("writing draft: %w", err)
	}
	return draft.Name(), nil
}

func validate(draft string) error {
	content, err := os.ReadFile(draft) //nolint: gosec
	if err != nil {
		return fmt.Errorf("reading draft: %w", err)
	}
	_, err = cfgfile.Parse(content)
	return err
}

// runEditor opens file in the user's editor and waits for it to exit. The
// editor is a shell command, so a value like "code --wait" works.
func runEditor(ctx context.Context, file string) error {
	cmd := editor() + ` "$1"`
	script, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return fmt.Errorf("parsing editor %q: %w", cmd, err)
	}
	runner, err := interp.New(
		interp.Params("--", file),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
	)
	if err != nil {
		return fmt.Errorf("creating shell: %w", err)
	}
	if err := runner.Run(ctx, script); err != nil {
		return fmt.Errorf("running editor %q: %w", cmd, err)
	}
	return nil
}

// editor names the user's editor the way git does: $VISUAL, then $EDITOR,
// then the platform's default.
func editor() string {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
