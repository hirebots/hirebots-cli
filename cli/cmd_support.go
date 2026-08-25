package main

// cmd_support.go — `hirebots support` commands.

import (
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
	prettyPrint(body)
	return nil
}

func runSupportShow(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/support/tickets/detail/" + args[0])
	if err != nil {
		return fmt.Errorf("fetching ticket: %w", err)
	}
	prettyPrint(body)
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
	prettyPrint(body)
	return nil
}