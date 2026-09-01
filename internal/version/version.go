// Package version exposes the panel's release version.
package version

// Version is overridden by release builds via -ldflags. The fallback keeps
// local/dev builds comparable to the upstream revision this fork started from.
var Version = "2.10.1"
