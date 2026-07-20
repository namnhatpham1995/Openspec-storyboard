// Package frontend exposes Storyboard's production web interface to the Go
// server. The generated dist directory is committed so every Go build embeds
// the exact same assets.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Dist contains the production Vite bundle, rooted at its public directory.
var Dist = mustSub(files, "dist")

func mustSub(source fs.FS, directory string) fs.FS {
	sub, err := fs.Sub(source, directory)
	if err != nil {
		panic(err)
	}
	return sub
}
