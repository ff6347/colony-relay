// ABOUTME: Tests for @mention parser
// ABOUTME: Validates extraction of @names, @all, @here from message bodies

package relay

import (
	"reflect"
	"testing"
)

func TestParseMentions(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		want     []string
		wantAll  bool
		wantHere bool
	}{
		{
			name: "no mentions",
			body: "hello world",
			want: nil,
		},
		{
			name: "single mention",
			body: "hello @agent-one",
			want: []string{"agent-one"},
		},
		{
			name: "multiple mentions",
			body: "@agent-one please talk to @agent-two",
			want: []string{"agent-one", "agent-two"},
		},
		{
			name: "mention at start",
			body: "@builder hello",
			want: []string{"builder"},
		},
		{
			name: "mention at end",
			body: "hello @builder",
			want: []string{"builder"},
		},
		{
			name: "mention with underscore",
			body: "hello @agent_one",
			want: []string{"agent_one"},
		},
		{
			name: "mention with numbers",
			body: "hello @agent123",
			want: []string{"agent123"},
		},
		{
			name:    "@all broadcast",
			body:    "@all hello everyone",
			want:    nil,
			wantAll: true,
		},
		{
			name:     "@here broadcast",
			body:     "@here anyone awake?",
			want:     nil,
			wantHere: true,
		},
		{
			name:    "mixed mentions and @all",
			body:    "@all and specifically @agent-one",
			want:    []string{"agent-one"},
			wantAll: true,
		},
		{
			name: "duplicate mentions",
			body: "@agent-one hello @agent-one",
			want: []string{"agent-one"},
		},
		{
			name: "email address not a mention",
			body: "contact me at test@example.com",
			want: nil,
		},
		{
			name: "mention followed by punctuation",
			body: "hello @agent-one, how are you?",
			want: []string{"agent-one"},
		},
		{
			name: "mention followed by period",
			body: "I agree with @agent-one.",
			want: []string{"agent-one"},
		},
		{
			name: "mention in parentheses",
			body: "someone (@agent-one) said this",
			want: []string{"agent-one"},
		},
		{
			name: "mention with colon after",
			body: "@agent-one: hello",
			want: []string{"agent-one"},
		},
		{
			name: "empty string",
			body: "",
			want: nil,
		},
		{
			name: "just @",
			body: "@ alone is not a mention",
			want: nil,
		},
		{
			name: "@ at end",
			body: "trailing @",
			want: nil,
		},
		{
			name:     "@all and @here together",
			body:     "@all @here wake up",
			wantAll:  true,
			wantHere: true,
		},
		{
			name: "case normalized to lowercase",
			body: "hello @AgentOne",
			want: []string{"agentone"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMentions(tt.body)

			if !reflect.DeepEqual(result.Names, tt.want) {
				t.Errorf("Names = %v, want %v", result.Names, tt.want)
			}
			if result.All != tt.wantAll {
				t.Errorf("All = %v, want %v", result.All, tt.wantAll)
			}
			if result.Here != tt.wantHere {
				t.Errorf("Here = %v, want %v", result.Here, tt.wantHere)
			}
		})
	}
}

func TestMentions_Contains(t *testing.T) {
	tests := []struct {
		name     string
		mentions Mentions
		target   string
		want     bool
	}{
		{
			name:     "direct mention",
			mentions: Mentions{Names: []string{"agent-one", "agent-two"}},
			target:   "agent-one",
			want:     true,
		},
		{
			name:     "not mentioned",
			mentions: Mentions{Names: []string{"agent-one"}},
			target:   "agent-two",
			want:     false,
		},
		{
			name:     "@all includes everyone",
			mentions: Mentions{All: true},
			target:   "anyone",
			want:     true,
		},
		{
			name:     "@here includes everyone",
			mentions: Mentions{Here: true},
			target:   "anyone",
			want:     true,
		},
		{
			name:     "empty mentions",
			mentions: Mentions{},
			target:   "agent-one",
			want:     false,
		},
		{
			name:     "case insensitive match",
			mentions: Mentions{Names: []string{"agent-one"}},
			target:   "AGENT-ONE",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mentions.Contains(tt.target); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}
