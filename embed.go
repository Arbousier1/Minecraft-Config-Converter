package mccassets

import "embed"

// Files embeds the built frontend and static image assets.
//
//go:embed web/dist web/static/images
var Files embed.FS
