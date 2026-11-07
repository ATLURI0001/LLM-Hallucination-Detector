package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

// GetStaticFS returns the embedded filesystem rooted at the static directory.
func GetStaticFS() fs.FS {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
