// Package mise holds intl's mise tasks (tasks.toml): go tool intl tasks writes them into an app, from the intl
// version in its go.mod.
package mise

import _ "embed"

// Tasks is tasks.toml.
//
//go:embed tasks.toml
var Tasks []byte
