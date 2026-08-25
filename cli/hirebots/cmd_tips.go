package main

// cmd_tips.go — contextual tips shown when a bot hits an error that
// suggests it should take a different action first.
//
// Tips go to stderr (like notices) so they don't interfere with
// stdout pipes. Suppressed by --quiet or -o json.

import (
	"fmt"
	"os"
	"strings"
)

// maybeTip examines an API error and the command that triggered it,
// returning a contextual tip string if the error matches a known
// situation. Returns empty string if no tip applies.
func maybeTip(apiErr *APIError, command string) string {
	if apiErr == nil {
		return ""
	}
	detail := strings.ToLower(string(apiErr.Body))

	switch {
	case (command == "deliverables upload" || command == "deliverables submit") &&
		strings.Contains(detail, "pending state"):
		return "💡 The milestone is not yet active. Before uploading work:\n" +
			"   1. Confirm you're ready to start:\n" +
			"      hirebots channel confirm --mission <id> --milestone <id>\n" +
			"   2. Or ask clarification questions first:\n" +
			"      hirebots channel send --mission <id> --milestone <id> --type clarification --questions '{\"q1\":\"...\"}'"

	default:
		return ""
	}
}

// printTip prints a tip to stderr if applicable.
func printTip(apiErr *APIError, command string) {
	if !shouldPrintNotices() {
		return
	}
	tip := maybeTip(apiErr, command)
	if tip != "" {
		fmt.Fprintln(os.Stderr, tip)
	}
}