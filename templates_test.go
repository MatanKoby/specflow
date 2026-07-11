package specflow

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbedManifestCoversAllTemplateFiles guards the //go:embed all:templates footgun. Without the
// all: prefix, Go silently drops dotfiles and dot-directories (.claude/, .cursor/, .github/,
// .agents/, .bob/) from the embedded tree — the Go analogue of the npm dropped-dotfile trap — so a
// fresh install would ship, say, without the Claude skills. This walks the on-disk templates/ tree
// and asserts every file is embedded byte-for-byte, and that at least one dot-path is among them so
// the guard is actually exercised.
func TestEmbedManifestCoversAllTemplateFiles(t *testing.T) {
	emb := Templates()
	var checked, dotPaths int

	err := filepath.WalkDir("templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Templates() subs the FS at "templates", so the embedded key is the path relative to it.
		rel, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		want, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		got, err := fs.ReadFile(emb, rel)
		if err != nil {
			t.Errorf("template %q is on disk but NOT embedded (missing go:embed all: coverage?): %v", rel, err)
			return nil
		}
		if !bytes.Equal(got, want) {
			t.Errorf("embedded template %q differs from the on-disk source", rel)
		}
		checked++
		for _, seg := range strings.Split(rel, "/") {
			if strings.HasPrefix(seg, ".") {
				dotPaths++
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk templates/: %v", err)
	}
	if checked == 0 {
		t.Fatal("no template files were walked — the test wiring is broken")
	}
	if dotPaths == 0 {
		t.Error("no dot-path templates were checked — the all: dotfile guard is not actually exercised")
	}
}
