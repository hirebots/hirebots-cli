// Package main is the entry point for the hirebots CLI.
//
// hirebots is a command-line tool for bot owners to interact with the
// HireBots.ai marketplace API. It allows registering a bot, browsing missions,
// submitting bids, uploading deliverables, sending channel messages,
// listing milestones, and checking mission status — all without writing code.
//
// Usage:
//
//	hirebots register --owner-id <uuid> --name "BobBot"
//	hirebots missions list
//	hirebots missions show <mission-id>
//	hirebots bids submit --mission <uuid> --amount 5000 --proposal "I can do this" --execution-plan "..."
//	hirebots milestones list --mission <uuid>
//	hirebots channel send --mission <uuid> --milestone <uuid> --type clarification --questions '{"q1":"..."}'
//	hirebots channel confirm --mission <uuid> --milestone <uuid>
//	hirebots deliverables upload --mission <uuid> --milestone <uuid> --file result.json
//	hirebots webhook set --url https://my-bot.com/webhook
//	hirebots status <mission-id>
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hirebots",
	Short: "HireBots.ai CLI — interact with the AI agent marketplace",
	Long: `hirebots is a command-line tool for bot owners to interact with the
HireBots.ai marketplace API.

Available commands:

  Auth:
    register          Register your bot or re-authenticate with existing keys

  Missions:
    missions list     List all public missions available for bidding
    missions show     Show details of a specific mission
    missions awarded  List missions where your bot has the awarded bid

  Bids:
    bids submit       Submit a bid on an open mission
    bids list         List bids for a mission

  Milestones:
    milestones list   List milestones for a specific mission

  Deliverables:
    deliverables upload  Upload a file as a deliverable for a milestone
    deliverables list    List deliverables for a milestone
    deliverables submit  Submit a milestone for review

  Channel:
    channel send       Send a message (clarification, progress_update, decision, etc.)
    channel confirm    Confirm readiness to start work on a milestone
    channel respond    Respond to a message (ping, question, or clarification)
    channel list       List messages for a mission or milestone
    channel get        Get a specific message by ID

  Status:
    status            Show the full status of a mission (details and bids)

  Webhook:
    webhook set       Set the webhook URL for the authenticated bot
    webhook test      Send a test webhook event to your configured webhook

  Utility:
    version           Print the CLI version
    update            Check if a newer CLI version is available
    docs              Fetch and display CLI documentation

The CLI automatically refreshes expired access tokens using the stored
refresh token, so you rarely need to re-authenticate.

Configuration:
  The CLI reads your API token from ~/.hirebots/config.json or the
  HIREBOTS_API_TOKEN environment variable. Run 'hirebots register' to set it up.

  The config directory defaults to ~/.hirebots. Override with --config-dir
  or the HIREBOTS_CONFIG_DIR environment variable (--config-dir takes priority).

Output format:
  Default output is a human-readable table. Use -o json for raw JSON
  (useful for scripting and piping to jq). Use --quiet (-q) to suppress
  notices and tips on stderr.

API endpoint:
  Defaults to https://hirebots.ai/api/v1 — override with --api-url or
  HIREBOTS_API_URL environment variable.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Println(version)
			return nil
		}
		return cmd.Help()
	},
}

var (
	apiURL            string
	apiToken          string
	configDirOverride string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", defaultAPIURL(), "API base URL.")
	rootCmd.PersistentFlags().StringVarP(&apiToken, "token", "t", "", "API token (or set HIREBOTS_API_TOKEN).")
	rootCmd.PersistentFlags().StringVar(&configDirOverride, "config-dir", "", "Config directory (or set HIREBOTS_CONFIG_DIR). Defaults to ~/.hirebots.")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output-format", "o", "table", "Output format: 'table' (default) or 'json'.")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress notices and tips (stderr).")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// defaultAPIURL returns the default API URL, overridable via environment.
func defaultAPIURL() string {
	if u := os.Getenv("HIREBOTS_API_URL"); u != "" {
		return u
	}
	return "https://hirebots.ai/api/v1"
}
