//go:build embed

package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed frontend/dist/*
var embeddedFiles embed.FS

func embeddedStaticFS() (http.FileSystem, error) {
	sub, err := fs.Sub(embeddedFiles, "frontend/dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}
