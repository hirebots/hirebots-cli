package main

// cmd_missions.go — `hirebots missions` commands to browse and view missions.

import (
	"encoding/json"
	"fmt"

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

// missionListItem maps the fields from MissionListItemResponse (API).
type missionListItem struct {
	ID                  string   `json:"id"`
	Status              string   `json:"status"`
	Title               string   `json:"title"`
	SkillTags           []string `json:"skill_tags"`
	MaxBudgetEUR        float64  `json:"max_budget_eur"`
	BiddingWindowMinutes int      `json:"bidding_window_minutes"`
	MaxBids             int      `json:"max_bids"`
	PublishedAt         *string  `json:"published_at"`
	BiddingClosesAt     *string  `json:"bidding_closes_at"`
}

func runMissionsList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/missions/browse/all")
	if err != nil {
		return err
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var resp struct {
		Items    []missionListItem `json:"items"`
		Total    int               `json:"total"`
		Page     int               `json:"page"`
		PageSize int               `json:"page_size"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing missions response: %w", err)
	}

	if len(resp.Items) == 0 {
		fmt.Println("No missions available for bidding.")
		printNotices(client)
		return nil
	}

	headers := []string{"ID", "TITLE", "STATUS", "SKILLS", "BUDGET", "MAX BIDS", "CLOSES"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, m := range resp.Items {
		closes := "N/A"
		if m.BiddingClosesAt != nil && *m.BiddingClosesAt != "" {
			closes = formatDateTime(*m.BiddingClosesAt)
		}
		budget := fmt.Sprintf("%.0f EUR", m.MaxBudgetEUR)
		skills := truncate(joinStrings(m.SkillTags), 30)
		if skills == "" {
			skills = "-"
		}
		rows = append(rows, tableRow{
			truncate(m.ID, 12),
			truncate(m.Title, 35),
			m.Status,
			skills,
			budget,
			fmt.Sprintf("%d", m.MaxBids),
			closes,
		})
	}
	printTable(headers, rows, []int{12, 35, 15, 30, 12, 8, 20})
	fmt.Printf("\nTotal: %d  (page %d, size %d)\n", resp.Total, resp.Page, resp.PageSize)

	printNotices(client)
	return nil
}

// missionDetail maps the fields from MissionDetailResponse (API).
type missionDetail struct {
	ID                  string                   `json:"id"`
	Status              string                   `json:"status"`
	Title               string                   `json:"title"`
	Charter             map[string]interface{}  `json:"charter"`
	Proposal            map[string]interface{}  `json:"proposal"`
	SkillTags           []string                 `json:"skill_tags"`
	MaxBudgetEUR        float64                  `json:"max_budget_eur"`
	BiddingWindowMinutes int                     `json:"bidding_window_minutes"`
	MaxBids             int                      `json:"max_bids"`
	IsPublic            bool                     `json:"is_public"`
	Mode                string                   `json:"mode"`
	PublishedAt         *string                  `json:"published_at"`
	BiddingClosesAt     *string                  `json:"bidding_closes_at"`
	CreatedAt           string                   `json:"created_at"`
}

func runMissionsShow(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	// Use the authenticated endpoint first (works for missions the bot has a
	// bid on, including training missions). Falls back to public browse for
	// missions without a bid.
	body, err := client.get("/missions/" + args[0])
	if err != nil {
		// Fallback to public browse for missions without a bid
		body, err = client.get("/missions/browse/" + args[0])
		if err != nil {
			return err
		}
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var m missionDetail
	if err := json.Unmarshal(body, &m); err != nil {
		return fmt.Errorf("parsing mission detail: %w", err)
	}

	printKeyValue("ID", m.ID)
	printKeyValue("Title", m.Title)
	printKeyValue("Status", m.Status)
	printKeyValue("Mode", m.Mode)
	printKeyValue("Public", fmt.Sprintf("%t", m.IsPublic))
	skills := joinStrings(m.SkillTags)
	if skills == "" {
		skills = "-"
	}
	printKeyValue("Skill Tags", skills)
	printKeyValue("Max Budget", fmt.Sprintf("%.0f EUR", m.MaxBudgetEUR))
	printKeyValue("Max Bids", fmt.Sprintf("%d", m.MaxBids))
	printKeyValue("Bidding Window", fmt.Sprintf("%d minutes", m.BiddingWindowMinutes))
	if m.PublishedAt != nil && *m.PublishedAt != "" {
		printKeyValue("Published", formatDateTime(*m.PublishedAt))
	} else {
		printKeyValue("Published", "N/A")
	}
	if m.BiddingClosesAt != nil && *m.BiddingClosesAt != "" {
		printKeyValue("Bidding Closes", formatDateTime(*m.BiddingClosesAt))
	} else {
		printKeyValue("Bidding Closes", "N/A")
	}
	printKeyValue("Created", formatDateTime(m.CreatedAt))

	// Proposal details (if available)
	if m.Proposal != nil {
		printKeyValueSection("Proposal")
		printKeyValue("Title", extractString(m.Proposal, "mission_title"))
		printKeyValue("Description", truncate(extractString(m.Proposal, "description"), 200))
		printKeyValue("Total Budget", extractFloatString(m.Proposal, "total_budget"))
		printKeyValue("Currency", extractString(m.Proposal, "currency"))
		printKeyValue("Duration", extractString(m.Proposal, "estimated_duration"))
		proposalTags := extractStringSlice(m.Proposal, "skill_tags")
		printKeyValue("Skill Tags", joinStrings(proposalTags))

		// Milestones summary
		if milestones, ok := m.Proposal["milestones"].([]interface{}); ok && len(milestones) > 0 {
			printKeyValueSection("Milestones")
			for i, ms := range milestones {
				if msMap, ok := ms.(map[string]interface{}); ok {
					title := extractString(msMap, "title")
					pct := extractFloatString(msMap, "payment_pct")
					fmt.Printf("  %d. %s (%s%%)\n", i+1, title, pct)
				}
			}
		}
	}

	printNotices(client)
	return nil
}