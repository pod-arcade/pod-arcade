//go:build !noembed

package main

import "embed"

//go:embed build
var httpStaticContent embed.FS
