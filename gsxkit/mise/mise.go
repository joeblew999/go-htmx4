// Package mise holds gsxkit's mise tasks (tasks.toml): go tool gsxkit tasks writes them into an app, from the gsxkit
// version in its go.mod.
package mise

import _ "embed"

// Tasks is tasks.toml.
//
//go:embed tasks.toml
var Tasks []byte
