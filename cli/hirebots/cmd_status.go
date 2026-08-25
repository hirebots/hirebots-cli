package main

// cmd_status.go — `hirebots status` command to check mission progress.

import (
	"encoding/json"
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

	if wantJSON() {
		fmt.Println("=== Mission ===")
		printJSON(missionBody)
		// Get bids
		bidsBody, err := client.get("/missions/" + missionID + "/bids")
		if err == nil {
			fmt.Println("=== Bids ===")
			printJSON(bidsBody)
		}
		return nil
	}

	// Table mode: show mission details
	var m missionDetail
	if err := json.Unmarshal(missionBody, &m); err != nil {
		// Can't parse — fall back to raw JSON
		fmt.Println("=== Mission ===")
		printJSON(missionBody)
	} else {
		printKeyValueSection("Mission")
		printKeyValue("ID", m.ID)
		printKeyValue("Title", m.Title)
		printKeyValue("Status", m.Status)
		skills := joinStrings(m.SkillTags)
		if skills == "" {
			skills = "-"
		}
		printKeyValue("Skill Tags", skills)
		printKeyValue("Max Budget", fmt.Sprintf("%.0f EUR", m.MaxBudgetEUR))
		if m.BiddingClosesAt != nil && *m.BiddingClosesAt != "" {
			printKeyValue("Bidding Closes", formatDateTime(*m.BiddingClosesAt))
		}
	}

	// Get bids
	bidsBody, err := client.get("/missions/" + missionID + "/bids")
	if err == nil {
		if wantJSON() {
			fmt.Println("=== Bids ===")
			printJSON(bidsBody)
		} else {
			printBidSummary(bidsBody)
		}
	}

	printNotices(client)
	return nil
}

// printBidSummary prints a compact summary of bids from the bids list response.
func printBidSummary(body []byte) {
	var resp struct {
		Items []struct {
			ID             string  `json:"id"`
			BotDisplayName string  `json:"bot_display_name"`
			TotalPriceEUR  string  `json:"total_price_eur"`
			Status         string  `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fmt.Println("=== Bids ===")
		printJSON(body)
		return
	}

	if len(resp.Items) == 0 {
		return
	}

	printKeyValueSection("Bids")
	headers := []string{"ID", "BOT", "AMOUNT", "STATUS"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, b := range resp.Items {
		rows = append(rows, tableRow{
			truncate(b.ID, 12),
			truncate(b.BotDisplayName, 20),
			b.TotalPriceEUR + " EUR",
			b.Status,
		})
	}
	printTable(headers, rows, []int{12, 20, 12, 15})
	fmt.Printf("\nTotal bids: %d\n", resp.Total)
}