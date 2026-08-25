package main

// cmd_support.go — `hirebots support` commands.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Manage support tickets",
}

var supportListCmd = &cobra.Command{
	Use:   "list [mission-id]",
	Short: "List support tickets for a mission",
	Args:  cobra.ExactArgs(1),
	RunE:  runSupportList,
}

var supportShowCmd = &cobra.Command{
	Use:   "show [ticket-id]",
	Short: "Show details of a support ticket (messages + attachments)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSupportShow,
}

var supportRespondCmd = &cobra.Command{
	Use:   "respond [ticket-id]",
	Short: "Respond to a support ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runSupportRespond,
}

var supportMessage string

func init() {
	supportRespondCmd.Flags().StringVarP(&supportMessage, "message", "m", "", "Message text (required).")
	_ = supportRespondCmd.MarkFlagRequired("message")

	supportCmd.AddCommand(supportListCmd)
	supportCmd.AddCommand(supportShowCmd)
	supportCmd.AddCommand(supportRespondCmd)
	rootCmd.AddCommand(supportCmd)
}

func runSupportList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/support/tickets/" + args[0])
	if err != nil {
		return fmt.Errorf("listing support tickets: %w", err)
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var resp struct {
		Items []struct {
			ID           string `json:"id"`
			TicketNumber string `json:"ticket_number"`
			Type         string `json:"type"`
			Status       string `json:"status"`
			Message      string `json:"message"`
			CreatedAt    string `json:"created_at"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		printJSON(body)
		return nil
	}

	if len(resp.Items) == 0 {
		fmt.Println("No support tickets found for this mission.")
		printNotices(client)
		return nil
	}

	headers := []string{"ID", "TICKET#", "TYPE", "STATUS", "MESSAGE", "CREATED"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, t := range resp.Items {
		rows = append(rows, tableRow{
			truncate(t.ID, 12),
			truncate(t.TicketNumber, 10),
			truncate(t.Type, 12),
			truncate(t.Status, 16),
			truncate(t.Message, 30),
			truncate(formatDateTime(t.CreatedAt), 16),
		})
	}
	printTable(headers, rows, []int{12, 10, 12, 16, 30, 16})
	fmt.Printf("\nTotal: %d\n", resp.Total)

	printNotices(client)
	return nil
}

func runSupportShow(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/support/tickets/detail/" + args[0])
	if err != nil {
		return fmt.Errorf("fetching ticket: %w", err)
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	// Ticket detail is complex enough to show as formatted JSON
	printJSON(body)
	printNotices(client)
	return nil
}

func runSupportRespond(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	payload := map[string]interface{}{
		"content": supportMessage,
	}
	body, err := client.post("/support/tickets/"+args[0]+"/messages", payload)
	if err != nil {
		return fmt.Errorf("responding to ticket: %w", err)
	}
	fmt.Println("✓ Response sent")
	if wantJSON() {
		printJSON(body)
	}
	printNotices(client)
	return nil
}