package main

// cmd_webhook.go — webhook utility commands for bots.

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Webhook utilities for bot integrations",
}

var webhookSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set the webhook URL for the authenticated bot",
	RunE:  runWebhookSet,
}

var webhookTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test webhook event to your configured bot webhook",
	RunE:  runWebhookTest,
}

var (
	webhookURL              string
	webhookTestEventType   string
	webhookTestPayloadFile string
)

func init() {
	webhookSetCmd.Flags().StringVarP(&webhookURL, "url", "u", "", "Webhook URL to set (required).")
	_ = webhookSetCmd.MarkFlagRequired("url")

	webhookTestCmd.Flags().StringVar(
		&webhookTestEventType,
		"event-type",
		"mission_published",
		"Event type to send in the test notification.",
	)
	webhookTestCmd.Flags().StringVar(
		&webhookTestPayloadFile,
		"payload-file",
		"",
		"Optional path to JSON payload file.",
	)

	webhookCmd.AddCommand(webhookSetCmd)
	webhookCmd.AddCommand(webhookTestCmd)
	rootCmd.AddCommand(webhookCmd)
}

func runWebhookSet(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)

	body, err := client.patch("/bots/me/webhook", map[string]string{
		"webhook_url": webhookURL,
	})
	if err != nil {
		return fmt.Errorf("setting webhook URL: %w", err)
	}

	fmt.Printf("✓ Webhook URL set to %s\n", webhookURL)
	prettyPrint(body)
	return nil
}

func runWebhookTest(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)

	// Resolve current authenticated user to route test event to this bot.
	meBody, err := client.get("/auth/me")
	if err != nil {
		return fmt.Errorf("fetching authenticated profile: %w", err)
	}

	var me struct {
		UserID   string `json:"user_id"`
		UserType string `json:"user_type"`
	}
	if err := jsonUnmarshal(meBody, &me); err != nil {
		return fmt.Errorf("parsing /auth/me response: %w", err)
	}
	if me.UserType != "bot" {
		return fmt.Errorf("webhook test requires bot authentication (current user_type=%s)", me.UserType)
	}
	if me.UserID == "" {
		return fmt.Errorf("missing user_id in /auth/me response")
	}

	payload := map[string]interface{}{
		"mission_title": "Webhook test event",
		"source":        "hirebots-cli",
	}
	if webhookTestPayloadFile != "" {
		raw, readErr := os.ReadFile(webhookTestPayloadFile)
		if readErr != nil {
			return fmt.Errorf("reading payload file: %w", readErr)
		}
		if unmarshalErr := json.Unmarshal(raw, &payload); unmarshalErr != nil {
			return fmt.Errorf("parsing payload file JSON: %w", unmarshalErr)
		}
	}

	req := map[string]interface{}{
		"event_type":     webhookTestEventType,
		"recipient_id":   me.UserID,
		"recipient_type": "bot",
		"payload":        payload,
	}

	resp, err := client.post("/notifications/send", req)
	if err != nil {
		return fmt.Errorf("sending webhook test notification: %w", err)
	}

	fmt.Println("Webhook test notification dispatched:")
	prettyPrint(resp)
	return nil
}