package main

// cmd_channel.go — `hirebots channel` commands for the mission channel (GAP-31).
// Replaces cmd_questions.go with a generalised channel supporting multiple
// message types: clarification, confirm_ready, progress_update, decision,
// client_note, client_question, client_ping.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Send and respond to messages in the mission channel",
	Long: `The mission channel is the communication channel between client and bot
during mission execution. All messages flow through the AI advisor (advocate model).

Bot commands:
  channel send       Send a message (clarification, progress_update)
  channel confirm    Confirm readiness to start a milestone
  channel respond    Respond to a client ping or question

Client commands (using a client API token):
  channel send       Send a message (decision, client_note, client_question, client_ping)
  channel respond    Respond to a clarification

Listing:
  channel list       List messages for a mission or milestone
  channel get        Get a specific message by ID`,
}

var channelSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message through the mission channel",
	RunE:  runChannelSend,
}

var channelConfirmCmd = &cobra.Command{
	Use:   "confirm",
	Short: "Confirm readiness to start work on a milestone (no questions needed)",
	RunE:  runChannelConfirm,
}

var channelRespondCmd = &cobra.Command{
	Use:   "respond",
	Short: "Respond to a message (clarification response, ping response, question response)",
	RunE:  runChannelRespond,
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List messages for a mission (optionally filtered by milestone)",
	RunE:  runChannelList,
}

var channelGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a specific message by ID",
	RunE:  runChannelGet,
}

var (
	chMissionID   string
	chMilestoneID string
	chMessageType string
	chContent     string
	chQuestions   string
	chMessageID   string
	chResponses   string
)

func init() {
	// send flags
	channelSendCmd.Flags().StringVarP(&chMissionID, "mission", "m", "", "Mission UUID (required).")
	channelSendCmd.Flags().StringVarP(&chMilestoneID, "milestone", "s", "", "Milestone UUID (optional for some types, required for clarification/decision/ping).")
	channelSendCmd.Flags().StringVarP(&chMessageType, "type", "", "", "Message type: clarification, confirm_ready, progress_update, decision, client_note, client_question, client_ping (required).")
	channelSendCmd.Flags().StringVarP(&chContent, "content", "c", "", "Message content text (required for most types).")
	channelSendCmd.Flags().StringVarP(&chQuestions, "questions", "q", "", `For clarification type only: JSON object of questions, e.g. '{"q1":"What format?","q2":"Deadline?"}'.`)
	_ = channelSendCmd.MarkFlagRequired("mission")
	_ = channelSendCmd.MarkFlagRequired("type")

	// confirm flags
	channelConfirmCmd.Flags().StringVarP(&chMissionID, "mission", "m", "", "Mission UUID (required).")
	channelConfirmCmd.Flags().StringVarP(&chMilestoneID, "milestone", "s", "", "Milestone UUID (required).")
	_ = channelConfirmCmd.MarkFlagRequired("mission")
	_ = channelConfirmCmd.MarkFlagRequired("milestone")

	// respond flags
	channelRespondCmd.Flags().StringVarP(&chMessageID, "message", "i", "", "Message UUID to respond to (required).")
	channelRespondCmd.Flags().StringVarP(&chContent, "content", "c", "", "Response content text (required for most types).")
	channelRespondCmd.Flags().StringVarP(&chResponses, "responses", "r", "", `For clarification response only: JSON object of responses, e.g. '{"q1":"Use FastAPI","q2":"No deadline"}'.`)
	_ = channelRespondCmd.MarkFlagRequired("message")

	// list flags
	channelListCmd.Flags().StringVarP(&chMissionID, "mission", "m", "", "Mission UUID (required).")
	channelListCmd.Flags().StringVarP(&chMilestoneID, "milestone", "s", "", "Milestone UUID (optional, filters by milestone).")
	_ = channelListCmd.MarkFlagRequired("mission")

	// get flags
	channelGetCmd.Flags().StringVarP(&chMessageID, "message", "i", "", "Message UUID (required).")
	_ = channelGetCmd.MarkFlagRequired("message")

	channelCmd.AddCommand(channelSendCmd)
	channelCmd.AddCommand(channelConfirmCmd)
	channelCmd.AddCommand(channelRespondCmd)
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelGetCmd)
	rootCmd.AddCommand(channelCmd)
}

func runChannelSend(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)

	body := map[string]interface{}{
		"mission_id": chMissionID,
		"type":       chMessageType,
		"content":    chContent,
	}
	if chMilestoneID != "" {
		body["milestone_id"] = chMilestoneID
	}

	// For clarification: parse questions JSON into a list of dicts
	if chMessageType == "clarification" {
		if chQuestions == "" {
			return fmt.Errorf("--questions is required for clarification type")
		}
		var qMap map[string]string
		if err := json.Unmarshal([]byte(chQuestions), &qMap); err != nil {
			return fmt.Errorf("parsing --questions JSON: %w", err)
		}
		qList := make([]map[string]string, 0, len(qMap))
		for k, v := range qMap {
			qList = append(qList, map[string]string{"id": k, "question": v})
		}
		body["questions"] = qList
	}

	resp, err := client.post("/channel/send", body)
	if err != nil {
		return fmt.Errorf("sending channel message: %w", err)
	}
	fmt.Println("✓ Message sent")
	prettyPrint(resp)
	return nil
}

func runChannelConfirm(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.post(
		"/channel/confirm-ready",
		map[string]interface{}{
			"mission_id":   chMissionID,
			"milestone_id": chMilestoneID,
		},
	)
	if err != nil {
		return fmt.Errorf("confirming readiness: %w", err)
	}
	fmt.Println("✓ Confirmed ready")
	prettyPrint(body)
	return nil
}

func runChannelRespond(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)

	body := map[string]interface{}{
		"message_id": chMessageID,
		"content":    chContent,
	}

	// If --responses is provided, parse as structured responses (for clarification)
	if chResponses != "" {
		var rMap map[string]string
		if err := json.Unmarshal([]byte(chResponses), &rMap); err != nil {
			return fmt.Errorf("parsing --responses JSON: %w", err)
		}
		rList := make([]map[string]string, 0, len(rMap))
		for k, v := range rMap {
			rList = append(rList, map[string]string{"id": k, "text": v})
		}
		body["responses"] = rList
	}

	resp, err := client.post("/channel/respond", body)
	if err != nil {
		return fmt.Errorf("responding to message: %w", err)
	}
	fmt.Println("✓ Response sent")
	prettyPrint(resp)
	return nil
}

func runChannelList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)

	path := "/channel/" + chMissionID
	if chMilestoneID != "" {
		path += "/" + chMilestoneID
	}

	body, err := client.get(path)
	if err != nil {
		return fmt.Errorf("listing channel messages: %w", err)
	}
	prettyPrint(body)
	return nil
}

func runChannelGet(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/channel/message/" + chMessageID)
	if err != nil {
		return fmt.Errorf("getting message: %w", err)
	}
	prettyPrint(body)
	return nil
}