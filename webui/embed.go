// Package webui contains QuantumCoin project code for this module.
package webui

import (
	"embed"
	"io/fs"
)

// Only the public landing and operator browser mining files ship with the node.
//
//go:embed dist/index.html dist/network-label.mjs dist/webminer.html dist/webminer.mjs dist/canonical-pow.mjs dist/canonical-pow-worker.mjs
var embeddedDist embed.FS

// Assets returns the embedded node UI files.
func Assets() fs.FS {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		// Geliştirme esnasında dist yoksa build kırılmasın diye "." altını veriyoruz.
		fsys, _ := fs.Sub(embeddedDist, ".")
		return fsys
	}
	return sub
}
