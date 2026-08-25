package main

// cmd_missions.go — `hirebots missions` commands to browse and view missions.

import (
	"github.com/spf13/cobra"
)

var missionsCmd = &cobra.Command{
	Use:   "missions",
	Short: "Browse and view missions on the marketplace",
}

var missionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all public missions available for bidding",
	RunE:  runMissionsList,
}

var missionsShowCmd = &cobra.Command{
	Use:   "show [mission-id]",
	Short: "Show details of a specific mission",
	Args:  cobra.ExactArgs(1),
	RunE:  runMissionsShow,
}

func init() {
	missionsCmd.AddCommand(missionsListCmd)
	missionsCmd.AddCommand(missionsShowCmd)
	rootCmd.AddCommand(missionsCmd)
}

func runMissionsList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/missions/browse/all")
	if err != nil {
		return err
	}
	prettyPrint(body)
	return nil
}

func runMissionsShow(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	// Use the authenticated endpoint, not the public browse. This works
	// for missions the bot has a bid on (including training missions,
	// which are not public). For public missions the bot doesn't have a
	// bid on, fall back to the public browse endpoint.
	body, err := client.get("/missions/" + args[0])
	if err != nil {
		// Fallback to public browse for missions without a bid
		body, err = client.get("/missions/browse/" + args[0])
		if err != nil {
			return err
		}
	}
	prettyPrint(body)
	return nil
}