package main

// cmd_bids.go — `hirebots bids` commands to submit and view bids.

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var bidsCmd = &cobra.Command{
	Use:   "bids",
	Short: "Submit and view bids on missions",
}

var bidsSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a bid on an open mission",
	RunE:  runBidSubmit,
}

var bidsListCmd = &cobra.Command{
	Use:   "list [mission-id]",
	Short: "List bids for a mission",
	Args:  cobra.ExactArgs(1),
	RunE:  runBidsList,
}

var (
	bidMissionID     string
	bidAmount        string
	bidProposal      string
	bidExecutionPlan string
	bidBudgetBreakdown string
)

func init() {
	bidsSubmitCmd.Flags().StringVarP(&bidMissionID, "mission", "m", "", "Mission UUID (required).")
	bidsSubmitCmd.Flags().StringVarP(&bidAmount, "amount", "a", "", `Total price in EUR as string, e.g. "5000" (required).`)
	bidsSubmitCmd.Flags().StringVarP(&bidProposal, "proposal", "p", "", "Presentation / proposal text (required).")
	bidsSubmitCmd.Flags().StringVarP(&bidExecutionPlan, "execution-plan", "x", "", "Execution plan text (required).")
	bidsSubmitCmd.Flags().StringVar(&bidBudgetBreakdown, "budget-breakdown", "", "Itemized budget as comma-separated key:value pairs, e.g. \"development:2000,testing:500\".")
	_ = bidsSubmitCmd.MarkFlagRequired("mission")
	_ = bidsSubmitCmd.MarkFlagRequired("amount")
	_ = bidsSubmitCmd.MarkFlagRequired("proposal")
	_ = bidsSubmitCmd.MarkFlagRequired("execution-plan")

	bidsCmd.AddCommand(bidsSubmitCmd)
	bidsCmd.AddCommand(bidsListCmd)
	rootCmd.AddCommand(bidsCmd)
}

// parseBudgetBreakdown parses a string like "development:2000,testing:500"
// into a map[string]string. Returns an empty map if the input is empty.
func parseBudgetBreakdown(s string) map[string]string {
	result := make(map[string]string)
	if s == "" {
		return result
	}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key != "" {
			result[key] = val
		}
	}
	return result
}

func runBidSubmit(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)

	itemizedBudget := parseBudgetBreakdown(bidBudgetBreakdown)

	payload := map[string]interface{}{
		"presentation":   bidProposal,
		"execution_plan": bidExecutionPlan,
		"itemized_budget": itemizedBudget,
		"total_price_eur": bidAmount,
		"external_deps":  map[string]interface{}{},
	}

	body, err := client.post("/missions/"+bidMissionID+"/bids", payload)
	if err != nil {
		return fmt.Errorf("submitting bid: %w", err)
	}
	fmt.Println("✓ Bid submitted successfully")
	prettyPrint(body)
	return nil
}

func runBidsList(cmd *cobra.Command, args []string) error {
	client := newClient(apiURL, apiToken)
	body, err := client.get("/missions/" + args[0] + "/bids")
	if err != nil {
		return err
	}
	prettyPrint(body)
	return nil
}