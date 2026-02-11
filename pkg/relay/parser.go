// ABOUTME: Extracts @mentions from message bodies
// ABOUTME: Handles @names, @all, and @here patterns

package relay

import (
	"regexp"
	"strings"
)

// Mentions holds the result of parsing @mentions from a message body.
type Mentions struct {
	Names []string // Specific @name mentions (excluding @all/@here)
	All   bool     // True if @all was mentioned
	Here  bool     // True if @here was mentioned
}

// mentionPattern matches @mentions that start at word boundary.
// Matches: @name, @entity-one, @entity_one, @entity123
// The negative lookbehind for alphanumeric prevents matching emails.
var mentionPattern = regexp.MustCompile(`(?:^|[^a-zA-Z0-9])@([a-zA-Z][a-zA-Z0-9_-]*)`)

// ParseMentions extracts all @mentions from a message body.
// All names are normalized to lowercase for case-insensitive matching.
func ParseMentions(body string) Mentions {
	var result Mentions
	seen := make(map[string]bool)

	matches := mentionPattern.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.ToLower(match[1])

		switch name {
		case "all":
			result.All = true
		case "here":
			result.Here = true
		default:
			if !seen[name] {
				seen[name] = true
				result.Names = append(result.Names, name)
			}
		}
	}

	return result
}

// Contains checks if a given name is mentioned (directly or via @all/@here).
// Comparison is case-insensitive.
func (m Mentions) Contains(name string) bool {
	if m.All || m.Here {
		return true
	}
	nameLower := strings.ToLower(name)
	for _, n := range m.Names {
		if n == nameLower {
			return true
		}
	}
	return false
}
