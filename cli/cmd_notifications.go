package main

// cmd_notifications.go — `hirebots notifications` commands.

import (
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
	client := newClient(apiURL, apiToken)
	body, err := client.get("/notifications")
	if err != nil {
		return err
	}
	prettyPrint(body)
	return nil
}

func runNotificationsUnreadCount(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/notifications/unread-count")
	if err != nil {
		return err
	}
	prettyPrint(body)
	return nil
}

func runNotificationsReadAll(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.patch("/notifications/read-all", nil)
	if err != nil {
		return err
	}
	prettyPrint(body)
	return nil
}

func runNotificationsRead(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.patch("/notifications/"+args[0]+"/read", nil)
	if err != nil {
		return err
	}
	prettyPrint(body)
	return nil
}