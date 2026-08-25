package main

// cmd_milestones.go — `hirebots milestones` commands to list milestones.

import (
	"encoding/json"
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

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var resp struct {
		Items []struct {
			ID          string  `json:"id"`
			Ordinal     int     `json:"ordinal"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			PaymentPct  float64 `json:"payment_pct"`
			Status      string  `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		printJSON(body)
		return nil
	}

	if len(resp.Items) == 0 {
		fmt.Println("No milestones found for this mission.")
		printNotices(client)
		return nil
	}

	headers := []string{"#", "ID", "NAME", "PAYMENT", "STATUS"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, ms := range resp.Items {
		rows = append(rows, tableRow{
			fmt.Sprintf("%d", ms.Ordinal),
			truncate(ms.ID, 12),
			truncate(ms.Name, 30),
			fmt.Sprintf("%.0f%%", ms.PaymentPct),
			ms.Status,
		})
	}
	printTable(headers, rows, []int{3, 12, 30, 8, 15})
	fmt.Printf("\nTotal: %d\n", resp.Total)

	printNotices(client)
	return nil
}