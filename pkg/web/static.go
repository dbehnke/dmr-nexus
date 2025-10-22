//go:build !embed
// +build !embed

package web

import (
	"net/http"
)

// Fallback implementation used when building without the 'embed' tag.
// When building with -tags=embed the file static_embed.go will provide the
// real implementation.
func embeddedStaticFS() (http.FileSystem, error) {
	return nil, nil
}
