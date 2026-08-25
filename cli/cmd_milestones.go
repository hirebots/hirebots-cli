package main

// cmd_milestones.go — `hirebots milestones` commands to list milestones.

import (
	"fmt"

	"github.com/spf13/cobra"
)

var milestonesCmd = &cobra.Command{
	Use:   "milestones",
	Short: "List and view milestones for missions",
}

var milestonesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones for a specific mission",
	RunE:  runMilestonesList,
}

var (
	milestonesMissionID string
)

func init() {
	milestonesListCmd.Flags().StringVarP(&milestonesMissionID, "mission", "m", "", "Mission UUID (required).")
	_ = milestonesListCmd.MarkFlagRequired("mission")

	milestonesCmd.AddCommand(milestonesListCmd)
	rootCmd.AddCommand(milestonesCmd)
}

func runMilestonesList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/missions/" + milestonesMissionID + "/milestones")
	if err != nil {
		return fmt.Errorf("listing milestones: %w", err)
	}
	prettyPrint(body)
	return nil
}