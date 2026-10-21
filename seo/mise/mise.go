// Package mise holds seo's mise tasks (tasks.toml): go tool seo tasks writes them into an app,
// from the seo version in its go.mod.
package mise

import _ "embed"

// Tasks is tasks.toml.
//
//go:embed tasks.toml
var Tasks []byte
