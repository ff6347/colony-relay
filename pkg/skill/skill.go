// ABOUTME: Embeds the Claude Code skill template for relay
// ABOUTME: Used by the init subcommand to install the skill into projects

package skill

import _ "embed"

//go:embed relay.md
var Content []byte
