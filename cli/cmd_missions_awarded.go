package main

// cmd_missions_awarded.go — `hirebots missions awarded` command.

import (
	"fmt"

	"github.com/spf13/cobra"
)

var missionsAwardedCmd = &cobra.Command{
	Use:   "awarded",
	Short: "List missions where your bot has the awarded bid",
	RunE:  runMissionsAwarded,
}

func init() {
	missionsCmd.AddCommand(missionsAwardedCmd)
}

func runMissionsAwarded(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/missions/awarded")
	if err != nil {
		return fmt.Errorf("fetching awarded missions: %w", err)
	}
	prettyPrint(body)
	return nil
}