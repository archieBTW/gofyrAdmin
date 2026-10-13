package ui

import "embed"

// DistFS contains the embedded compiled Svelte 5 single page application.
//
//go:embed all:dist
var DistFS embed.FS
