package main

// cmd_update.go — `hirebots update` command to check for CLI updates.
//
// Calls the public GET /cli/version endpoint and compares the returned
// latest_version / min_version against the current build version.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check if a newer CLI version is available",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// cliVersionResponse is the shape returned by GET /cli/version.
type cliVersionResponse struct {
	LatestVersion  string `json:"latest_version"`
	MinVersion     string `json:"min_version"`
	DownloadBaseURL string `json:"download_base_url"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	skipNoticeForCommand = true
	// Public endpoint — no auth token needed.
	client := newClient(apiURL, "")
	body, err := client.get("/cli/version")
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	var info cliVersionResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return fmt.Errorf("parsing version response: %w", err)
	}

	current := version
	latest := info.LatestVersion
	minV := info.MinVersion

	fmt.Printf("Current version: %s\n", current)
	fmt.Printf("Latest version:  %s\n", latest)

	if latest != "" && compareVersions(current, latest) < 0 {
		fmt.Println()
		fmt.Println("A newer version is available!")
		fmt.Println("Update with:")
		fmt.Println("  curl -fsSL https://hirebots.ai/install.sh | sh")
	}

	if minV != "" && compareVersions(current, minV) < 0 {
		fmt.Println()
		fmt.Printf("⚠️  WARNING: Your CLI version (%s) is older than the minimum supported version (%s).\n", current, minV)
		fmt.Println("   Some features may not work correctly. Please update as soon as possible.")
	}

	if (latest == "" || compareVersions(current, latest) >= 0) &&
		(minV == "" || compareVersions(current, minV) >= 0) {
		fmt.Println()
		fmt.Printf("✓ You're running the latest version (%s)\n", current)
	}

	return nil
}

// compareVersions compares two semantic version strings (e.g. "0.2.0").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Non-numeric or malformed segments are treated as 0. A "dev" version is
// considered lower than any concrete release.
func compareVersions(a, b string) int {
	if a == "dev" {
		return -1
	}
	if b == "dev" {
		return 1
	}

	aParts := parseSemver(a)
	bParts := parseSemver(b)

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		av, bv := 0, 0
		if i < len(aParts) {
			av = aParts[i]
		}
		if i < len(bParts) {
			bv = bParts[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// parseSemver extracts the numeric segments from a version string like
// "0.2.0" or "v1.2.3-rc1" into a slice of ints.
func parseSemver(v string) []int {
	// Strip a leading 'v'.
	v = strings.TrimPrefix(v, "v")
	// Take only the part before any '-' (pre-release suffix).
	if idx := strings.Index(v, "-"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			} else {
				break
			}
		}
		nums = append(nums, n)
	}
	return nums
}