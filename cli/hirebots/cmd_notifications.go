package main

// cmd_notifications.go — `hirebots notifications` commands.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var notificationsCmd = &cobra.Command{
	Use:   "notifications",
	Short: "View and manage your notifications",
}

var notificationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your notifications (newest first)",
	RunE:  runNotificationsList,
}

var notificationsUnreadCmd = &cobra.Command{
	Use:   "unread-count",
	Short: "Show count of unread notifications",
	RunE:  runNotificationsUnreadCount,
}

var notificationsReadAllCmd = &cobra.Command{
	Use:   "read-all",
	Short: "Mark all notifications as read",
	RunE:  runNotificationsReadAll,
}

var notificationsReadCmd = &cobra.Command{
	Use:   "read [notification-id]",
	Short: "Mark a single notification as read",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotificationsRead,
}

func init() {
	notificationsCmd.AddCommand(notificationsListCmd)
	notificationsCmd.AddCommand(notificationsUnreadCmd)
	notificationsCmd.AddCommand(notificationsReadAllCmd)
	notificationsCmd.AddCommand(notificationsReadCmd)
	rootCmd.AddCommand(notificationsCmd)
}

func runNotificationsList(cmd *cobra.Command, args []string) error {
	skipNoticeForCommand = true
	client := newClient(apiURL, apiToken)
	body, err := client.get("/notifications")
	if err != nil {
		return err
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var resp struct {
		Items []struct {
			ID        string `json:"id"`
			EventType string `json:"event_type"`
			Read      bool   `json:"read"`
			CreatedAt string `json:"created_at"`
			Payload   map[string]interface{} `json:"payload"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		printJSON(body)
		return nil
	}

	if len(resp.Items) == 0 {
		fmt.Println("No notifications found.")
		return nil
	}

	headers := []string{"ID", "EVENT", "READ", "CREATED"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, n := range resp.Items {
		read := "no"
		if n.Read {
			read = "yes"
		}
		rows = append(rows, tableRow{
			truncate(n.ID, 12),
			truncate(n.EventType, 25),
			read,
			truncate(formatDateTime(n.CreatedAt), 16),
		})
	}
	printTable(headers, rows, []int{12, 25, 5, 16})
	fmt.Printf("\nTotal: %d\n", resp.Total)
	return nil
}

func runNotificationsUnreadCount(cmd *cobra.Command, args []string) error {
	skipNoticeForCommand = true
	client := newClient(apiURL, apiToken)
	body, err := client.get("/notifications/unread-count")
	if err != nil {
		return err
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var resp struct {
		UnreadCount int `json:"unread_count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		printJSON(body)
		return nil
	}
	fmt.Printf("Unread notifications: %d\n", resp.UnreadCount)
	return nil
}

func runNotificationsReadAll(cmd *cobra.Command, args []string) error {
	skipNoticeForCommand = true
	client := newClient(apiURL, apiToken)
	body, err := client.patch("/notifications/read-all", nil)
	if err != nil {
		return err
	}
	if wantJSON() {
		printJSON(body)
	} else {
		fmt.Println("✓ All notifications marked as read")
	}
	return nil
}

func runNotificationsRead(cmd *cobra.Command, args []string) error {
	skipNoticeForCommand = true
	client := newClient(apiURL, apiToken)
	body, err := client.patch("/notifications/"+args[0]+"/read", nil)
	if err != nil {
		return err
	}
	if wantJSON() {
		printJSON(body)
	} else {
		fmt.Println("✓ Notification marked as read")
	}
	return nil
}