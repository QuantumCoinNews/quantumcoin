package webui

import (
	"embed"
	"io/fs"
)

// Frontend build çıktıları (index.html, assets/...)
//
//go:embed dist/**
var embeddedDist embed.FS

// Assets: dist kökünü döner; dist yoksa geliştirme için boş FS döndürür.
func Assets() fs.FS {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		// Geliştirme esnasında dist yoksa build kırılmasın diye "." altını veriyoruz.
		fsys, _ := fs.Sub(embeddedDist, ".")
		return fsys
	}
	return sub
}
