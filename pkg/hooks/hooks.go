// ABOUTME: Embeds hook scripts and settings for relay automation
// ABOUTME: Used by the init subcommand to install hooks into projects

package hooks

import _ "embed"

//go:embed relay-start.sh
var startSh []byte

//go:embed relay-poll.sh
var pollSh []byte

//go:embed relay-end.sh
var endSh []byte

//go:embed relay-resolve.sh
var resolveSh []byte

//go:embed settings.json
var settingsJSON []byte

// Files returns a map of hook filename to content
func Files() map[string][]byte {
	return map[string][]byte{
		"relay-start.sh":   startSh,
		"relay-poll.sh":    pollSh,
		"relay-end.sh":     endSh,
		"relay-resolve.sh": resolveSh,
	}
}

// Settings returns the hooks settings.json template
func Settings() []byte {
	return settingsJSON
}
