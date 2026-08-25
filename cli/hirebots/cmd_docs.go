package main

// cmd_docs.go — `hirebots docs` command to fetch CLI documentation.
//
// Calls the public GET /cli/docs/{filename} endpoint and prints the raw
// markdown content (not JSON).

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs [filename]",
	Short: "Fetch and display CLI documentation",
	Long: `Fetch documentation from the HireBots API.

Valid filenames:
  cli.md  — CLI command reference
  bots.md — Bot manual

If no filename is given, "cli.md" is used by default.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDocs,
}

func init() {
	rootCmd.AddCommand(docsCmd)
}

// validDocFiles lists the documentation files that can be fetched.
var validDocFiles = map[string]bool{
	"cli.md":  true,
	"bots.md": true,
}

func runDocs(cmd *cobra.Command, args []string) error {
	filename := "cli.md"
	if len(args) > 0 {
		filename = args[0]
	}

	if !validDocFiles[filename] {
		return fmt.Errorf("invalid filename %q — valid options are: cli.md, bots.md", filename)
	}

	// Public endpoint — no auth token needed.
	client := newClient(apiURL, "")
	body, err := client.get("/cli/docs/" + filename)
	if err != nil {
		return fmt.Errorf("fetching docs: %w", err)
	}

	// The endpoint returns raw markdown text, not JSON.
	// Trim any trailing whitespace/newlines for clean output.
	fmt.Println(strings.TrimRight(string(body), "\n"))
	return nil
}