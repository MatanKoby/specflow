// Package specflow embeds the protocol templates the CLI scaffolds into a host repo. The embed
// lives at the module root because a //go:embed directive cannot reference parent directories — so
// the templates tree (a sibling of this file) must be embedded from here.
package specflow

import (
	"embed"
	"io/fs"
)

// all: is mandatory so dotfiles and dot-dirs (.claude/, .cursor/, .github/, .agents/, .bob/,
// .spec-batch.json) are embedded — the Go analogue of the npm dropped-dotfile footgun.
//
//go:embed all:templates
var templatesFS embed.FS

// Templates returns the embedded templates tree rooted at the templates directory, so paths read as
// "base/AGENTS.md", "agents/claude/CLAUDE.md", and so on.
func Templates() fs.FS {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}
