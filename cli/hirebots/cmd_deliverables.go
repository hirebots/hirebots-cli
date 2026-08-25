package main

// cmd_deliverables.go — `hirebots deliverables` commands to upload work.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var deliverablesCmd = &cobra.Command{
	Use:   "deliverables",
	Short: "Upload and view deliverables for milestones",
}

var deliverablesUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file as a deliverable for a milestone",
	RunE:  runDeliverableUpload,
}

var deliverablesListCmd = &cobra.Command{
	Use:   "list [mission-id] [milestone-id]",
	Short: "List deliverables for a milestone",
	Args:  cobra.ExactArgs(2),
	RunE:  runDeliverablesList,
}

var deliverablesSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a milestone for review (mark deliverables complete)",
	RunE:  runDeliverableSubmit,
}

var (
	delMissionID   string
	delMilestoneID string
	delFile        string
	delLabel       string

	// deliverables submit flags
	delSubmitMissionID   string
	delSubmitMilestoneID string
)

func init() {
	deliverablesUploadCmd.Flags().StringVarP(&delMissionID, "mission", "m", "", "Mission UUID (required).")
	deliverablesUploadCmd.Flags().StringVarP(&delMilestoneID, "milestone", "s", "", "Milestone UUID (required).")
	deliverablesUploadCmd.Flags().StringVarP(&delFile, "file", "f", "", "Path to file to upload (required).")
	deliverablesUploadCmd.Flags().StringVarP(&delLabel, "label", "l", "", "Version label (e.g. 'v1.0').")
	_ = deliverablesUploadCmd.MarkFlagRequired("mission")
	_ = deliverablesUploadCmd.MarkFlagRequired("milestone")
	_ = deliverablesUploadCmd.MarkFlagRequired("file")

	deliverablesSubmitCmd.Flags().StringVarP(&delSubmitMissionID, "mission", "m", "", "Mission UUID (required).")
	deliverablesSubmitCmd.Flags().StringVarP(&delSubmitMilestoneID, "milestone", "s", "", "Milestone UUID (required).")
	_ = deliverablesSubmitCmd.MarkFlagRequired("mission")
	_ = deliverablesSubmitCmd.MarkFlagRequired("milestone")

	deliverablesCmd.AddCommand(deliverablesUploadCmd)
	deliverablesCmd.AddCommand(deliverablesListCmd)
	deliverablesCmd.AddCommand(deliverablesSubmitCmd)
	rootCmd.AddCommand(deliverablesCmd)
}

func runDeliverableUpload(cmd *cobra.Command, args []string) error {
	// Read and base64-encode the file
	data, err := os.ReadFile(delFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	fileB64 := base64.StdEncoding.EncodeToString(data)
	filename := filepath.Base(delFile)

	if delLabel == "" {
		delLabel = filename
	}

	client := newClient(apiURL, apiToken)
	payload := map[string]interface{}{
		"mission_id":   delMissionID,
		"milestone_id": delMilestoneID,
		"version_label": delLabel,
		"filename":     filename,
		"file_b64":     fileB64,
	}
	body, err := client.post("/deliverables/upload", payload)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			printTip(apiErr, "deliverables upload")
		}
		return fmt.Errorf("uploading deliverable: %w", err)
	}
	fmt.Printf("✓ Uploaded %s (%d bytes) as deliverable\n", filename, len(data))
	if wantJSON() {
		printJSON(body)
	}
	printNotices(client)
	return nil
}

func runDeliverablesList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/missions/" + args[0] + "/milestones/" + args[1] + "/deliverables")
	if err != nil {
		return err
	}

	if wantJSON() {
		printJSON(body)
		return nil
	}

	var resp struct {
		Items []struct {
			ID           string `json:"id"`
			Filename     string `json:"filename"`
			VersionLabel string `json:"version_label"`
			VersionNumber int   `json:"version_number"`
			FileSizeBytes int64 `json:"file_size_bytes"`
			CreatedAt    string `json:"created_at"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		printJSON(body)
		return nil
	}

	if len(resp.Items) == 0 {
		fmt.Println("No deliverables found for this milestone.")
		printNotices(client)
		return nil
	}

	headers := []string{"ID", "FILENAME", "VERSION", "SIZE", "CREATED"}
	rows := make([]tableRow, 0, len(resp.Items))
	for _, d := range resp.Items {
		rows = append(rows, tableRow{
			truncate(d.ID, 12),
			truncate(d.Filename, 24),
			fmt.Sprintf("v%d", d.VersionNumber),
			humanSize(d.FileSizeBytes),
			truncate(formatDateTime(d.CreatedAt), 16),
		})
	}
	printTable(headers, rows, []int{12, 24, 8, 10, 16})
	fmt.Printf("\nTotal: %d\n", resp.Total)

	printNotices(client)
	return nil
}

func runDeliverableSubmit(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	path := "/deliverables/missions/" + delSubmitMissionID + "/milestones/" + delSubmitMilestoneID + "/submit"
	body, err := client.post(path, nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			printTip(apiErr, "deliverables submit")
		}
		return fmt.Errorf("submitting milestone: %w", err)
	}
	fmt.Println("✓ Milestone submitted for review")
	if wantJSON() {
		printJSON(body)
	}
	printNotices(client)
	return nil
}