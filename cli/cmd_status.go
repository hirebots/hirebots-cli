package main

// cmd_status.go — `hirebots status` command to check mission progress.

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [mission-id]",
	Short: "Show the full status of a mission (details and bids)",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	missionID := args[0]

	// Use authenticated endpoint first (works for missions the bot has a
	// bid on, including training missions). Falls back to public browse
	// for missions without a bid.
	missionBody, err := client.get("/missions/" + missionID)
	if err != nil {
		missionBody, err = client.get("/missions/browse/" + missionID)
		if err != nil {
			return fmt.Errorf("fetching mission: %w", err)
		}
	}
	fmt.Println("=== Mission ===")
	prettyPrint(missionBody)

	// Get bids
	bidsBody, err := client.get("/missions/" + missionID + "/bids")
	if err == nil {
		fmt.Println("\n=== Bids ===")
		prettyPrint(bidsBody)
	}

	// Deliverables endpoint may 404 for non-awarded missions, so we skip it here.
	// Use `hirebots deliverables list` explicitly when needed.

	return nil
}