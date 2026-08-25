package main

// cmd_missions_awarded.go — `hirebots missions awarded` command.

import (
	"encoding/json"
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

	if wantJSON() {
		printJSON(body)
		return nil
	}

	// The awarded endpoint may return a list or a single mission.
	// Try to parse as a list response first.
	var resp struct {
		Items []missionListItem `json:"items"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		// Fall back to JSON pretty-print if we can't parse the structure
		printJSON(body)
		return nil
	}

	if len(resp.Items) == 0 {
		fmt.Println("No awarded missions found.")
		printNotices(client)
		return nil
	}

	headers := []string{"ID", "TITLE", "STATUS", "SKILLS", "BUDGET", "MAX BIDS"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, m := range resp.Items {
		skills := truncate(joinStrings(m.SkillTags), 30)
		if skills == "" {
			skills = "-"
		}
		rows = append(rows, tableRow{
			truncate(m.ID, 12),
			truncate(m.Title, 35),
			m.Status,
			skills,
			fmt.Sprintf("%.0f EUR", m.MaxBudgetEUR),
			fmt.Sprintf("%d", m.MaxBids),
		})
	}
	printTable(headers, rows, []int{12, 35, 15, 30, 12, 8})
	fmt.Printf("\nTotal: %d\n", resp.Total)

	printNotices(client)
	return nil
}