// Package skills holds the agent skills that ship with intl (as gsx ships skills/); go tool intl skill installs them
// into an app's .claude/skills.
package skills

import "embed"

// FS is every skill directory: intl-translate/SKILL.md.
//
//go:embed intl-translate
var FS embed.FS
