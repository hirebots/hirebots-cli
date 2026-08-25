package main

// cmd_support_attachments.go — `hirebots support attachments` commands.
//
// Support ticket attachments are NOT encrypted (unlike mission attachments),
// so there is no crypto layer here — just base64 for upload and raw bytes
// for download.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var supportAttachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "List, upload, download, and promote support ticket attachments",
}

var supportAttachmentsListCmd = &cobra.Command{
	Use:   "list <ticket-id>",
	Short: "List attachments for a support ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runSupportAttachmentsList,
}

var supportAttachmentsUploadCmd = &cobra.Command{
	Use:   "upload <ticket-id> <file-path>",
	Short: "Upload a file attachment to a support ticket",
	Args:  cobra.ExactArgs(2),
	RunE:  runSupportAttachmentsUpload,
}

var supportAttachmentsDownloadCmd = &cobra.Command{
	Use:   "download <attachment-id>",
	Short: "Download a support ticket attachment",
	Args:  cobra.ExactArgs(1),
	RunE:  runSupportAttachmentsDownload,
}

var supportAttachmentsPromoteCmd = &cobra.Command{
	Use:   "promote <attachment-id>",
	Short: "Promote a ticket attachment to a deliverable version",
	Args:  cobra.ExactArgs(1),
	RunE:  runSupportAttachmentsPromote,
}

var supportAttachmentOutput string

func init() {
	supportAttachmentsDownloadCmd.Flags().StringVarP(&supportAttachmentOutput, "output", "o", "", "Output file path (default: attachment_<id>.bin)")

	supportAttachmentsCmd.AddCommand(supportAttachmentsListCmd)
	supportAttachmentsCmd.AddCommand(supportAttachmentsUploadCmd)
	supportAttachmentsCmd.AddCommand(supportAttachmentsDownloadCmd)
	supportAttachmentsCmd.AddCommand(supportAttachmentsPromoteCmd)
	supportCmd.AddCommand(supportAttachmentsCmd)
}

// ticketAttachment mirrors the JSON returned by GET /support/tickets/{id}/attachments.
type ticketAttachment struct {
	ID            string   `json:"id"`
	TicketID      string   `json:"ticket_id"`
	UploadedBy    string   `json:"uploaded_by"`
	Filename      string   `json:"filename"`
	FileSizeBytes int64    `json:"file_size_bytes"`
	FileTypes     []string `json:"file_types"`
	CreatedAt     string   `json:"created_at"`
}

func runSupportAttachmentsList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	ticketID := args[0]

	body, err := client.get("/support/tickets/" + ticketID + "/attachments")
	if err != nil {
		return fmt.Errorf("listing ticket attachments: %w", err)
	}

	var items []ticketAttachment
	if err := json.Unmarshal(body, &items); err != nil {
		return fmt.Errorf("parsing attachments response: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("No attachments found for this ticket.")
		return nil
	}

	fmt.Printf("%-10s  %-24s  %-10s  %-16s  %s\n", "ID", "FILENAME", "SIZE", "UPLOADED_BY", "UPLOADED")
	fmt.Println(strings.Repeat("-", 80))
	for _, a := range items {
		fmt.Printf("%-10s  %-24s  %-10s  %-16s  %s\n",
			shortID(a.ID, 8),
			truncate(a.Filename, 24),
			humanSize(a.FileSizeBytes),
			truncate(a.UploadedBy, 16),
			formatDate(a.CreatedAt),
		)
	}
	fmt.Printf("\nTotal: %d\n", len(items))
	return nil
}

func runSupportAttachmentsUpload(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	ticketID := args[0]
	filePath := args[1]

	// Read the file from disk.
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filePath, err)
	}

	filename := filepath.Base(filePath)

	// Base64-encode the file content.
	fileB64 := base64.StdEncoding.EncodeToString(fileData)

	payload := map[string]string{
		"filename": filename,
		"file_b64": fileB64,
	}

	body, err := client.post("/support/tickets/"+ticketID+"/attachments", payload)
	if err != nil {
		return fmt.Errorf("uploading attachment: %w", err)
	}

	var resp ticketAttachment
	if err := json.Unmarshal(body, &resp); err != nil {
		// Non-fatal — print raw response.
		printJSON(body)
		return nil
	}

	fmt.Printf("✓ Uploaded %s (%d bytes) — attachment ID: %s\n", resp.Filename, resp.FileSizeBytes, resp.ID)
	return nil
}

func runSupportAttachmentsDownload(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	attachmentID := args[0]

	body, err := client.get("/support/attachments/" + attachmentID + "/download")
	if err != nil {
		return fmt.Errorf("downloading attachment: %w", err)
	}

	// Determine output filename.
	outName := supportAttachmentOutput
	if outName == "" {
		outName = "attachment_" + shortID(attachmentID, 8) + ".bin"
	}

	if err := os.WriteFile(outName, body, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Printf("✓ Saved %d bytes to %s\n", len(body), outName)
	return nil
}

func runSupportAttachmentsPromote(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	attachmentID := args[0]

	body, err := client.post("/support/attachments/"+attachmentID+"/promote", nil)
	if err != nil {
		return fmt.Errorf("promoting attachment: %w", err)
	}

	fmt.Println("✓ Attachment promoted to deliverable")
	if wantJSON() {
		printJSON(body)
	}
	return nil
}