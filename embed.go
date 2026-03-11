package mccassets

import "embed"

// Files embeds the existing frontend so the Go server can serve a single binary.
//
//go:embed web/templates/index.html web/static
var Files embed.FS
