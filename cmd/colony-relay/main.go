// ABOUTME: Entry point for the colony-relay CLI
// ABOUTME: Dispatches subcommands: start, say, hear, init, status

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	args := os.Args[2:]

	var exitCode int
	switch subcmd {
	case "start":
		exitCode = runStart(args)
	case "say":
		exitCode = runSay(args)
	case "hear":
		exitCode = runHear(args)
	case "init":
		exitCode = runInit(args)
	case "status":
		exitCode = runStatus(args)
	case "-h", "--help", "help":
		printUsage()
		exitCode = 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", subcmd)
		printUsage()
		exitCode = 1
	}

	os.Exit(exitCode)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `colony-relay - message relay for agent communication

Usage:
  colony-relay <command> [flags]

Commands:
  init     Initialize relay in current project
  start    Start the relay server
  say      Send a message
  hear     Receive messages
  status   Check relay status

Run 'colony-relay <command> --help' for details on each command.
`)
}
