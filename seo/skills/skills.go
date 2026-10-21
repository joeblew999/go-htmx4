// Package skills holds the agent skill that ships with seo (as intl ships intl-translate); go tool seo skill installs
// it into an app's .claude/skills.
package skills

import "embed"

// FS is every skill directory: seo/SKILL.md.
//
//go:embed seo
var FS embed.FS
