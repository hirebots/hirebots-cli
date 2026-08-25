package main

// cmd_notices.go — background notices shown after each command.
//
// Checks unread notifications count and CLI version freshness.
// Results are cached in ~/.hirebots/.cache.json with a 5-minute TTL
// to avoid hammering the API on every command invocation.
//
// Notices go to stderr so they don't interfere with stdout pipes.
// Suppressed by --quiet or -o json.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const noticeCacheTTL = 5 * time.Minute

// noticeCache is the on-disk cache structure.
type noticeCache struct {
	LastUnreadCheck  time.Time `json:"last_unread_check"`
	UnreadCount      int       `json:"unread_count"`
	LastVersionCheck time.Time `json:"last_version_check"`
	LatestVersion    string    `json:"latest_version"`
	MinVersion       string    `json:"min_version"`
}

// cachePath returns the path to the notice cache file.
func cachePath() string {
	return filepath.Join(configDir(), ".cache.json")
}

// loadCache reads the notice cache from disk. Returns zero-value cache
// if the file doesn't exist or can't be parsed.
func loadCache() noticeCache {
	var c noticeCache
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

// saveCache writes the notice cache to disk.
func saveCache(c noticeCache) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cachePath(), data, 0o600)
}

// checkUnreadCount fetches the unread notification count, using cache.
func checkUnreadCount(client *Client) int {
	c := loadCache()
	if !c.LastUnreadCheck.IsZero() && time.Since(c.LastUnreadCheck) < noticeCacheTTL {
		return c.UnreadCount
	}

	// Fresh fetch
	body, err := client.get("/notifications/unread-count")
	if err != nil {
		// API error — use cached value if available, else 0
		return c.UnreadCount
	}

	var resp struct {
		UnreadCount int `json:"unread_count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return c.UnreadCount
	}

	c.UnreadCount = resp.UnreadCount
	c.LastUnreadCheck = time.Now()
	saveCache(c)
	return resp.UnreadCount
}

// checkVersionFreshness fetches the latest CLI version, using cache.
// Returns (latestVersion, minVersion) — empty strings if unknown.
func checkVersionFreshness(client *Client) (string, string) {
	c := loadCache()
	if !c.LastVersionCheck.IsZero() && time.Since(c.LastVersionCheck) < noticeCacheTTL {
		return c.LatestVersion, c.MinVersion
	}

	// Fresh fetch — public endpoint, no auth needed
	body, err := client.get("/cli/version")
	if err != nil {
		return c.LatestVersion, c.MinVersion
	}

	var info cliVersionResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return c.LatestVersion, c.MinVersion
	}

	c.LatestVersion = info.LatestVersion
	c.MinVersion = info.MinVersion
	c.LastVersionCheck = time.Now()
	saveCache(c)
	return info.LatestVersion, info.MinVersion
}

// isNotificationCommand returns true if the current command is a
// notifications command — in that case we skip the cached notice
// because the user is explicitly checking notifications.
func isNotificationCommand() bool {
	// We check this via the command path. Since cobra doesn't give us
	// easy access to the full path from inside RunE, we use a simple
	// heuristic: the caller sets a flag before calling printNotices.
	return skipNoticeForCommand
}

// skipNoticeForCommand is set by notification commands to avoid
// redundant notices when the user is explicitly reading notifications.
var skipNoticeForCommand bool

// printNoticesImpl is the real implementation of printNotices.
// It checks unread notifications and CLI version, printing any
// pending notices to stderr.
func printNoticesImpl(client *Client) {
	if !shouldPrintNotices() {
		return
	}
	if skipNoticeForCommand {
		return
	}

	notices := []string{}

	// Check unread notifications (only if we have a token)
	if apiToken != "" {
		unread := checkUnreadCount(client)
		if unread > 0 {
			notices = append(notices, fmt.Sprintf(
				"─── %d unread notification(s) — run `hirebots notifications list` to view",
				unread,
			))
		}
	}

	// Check CLI version (public endpoint, always available)
	latest, minV := checkVersionFreshness(client)
	current := version
	if latest != "" && compareVersions(current, latest) < 0 {
		notices = append(notices, fmt.Sprintf(
			"─── CLI update available: %s → %s — run `curl -fsSL https://hirebots.ai/install.sh | sh` to update",
			current, latest,
		))
	}
	if minV != "" && compareVersions(current, minV) < 0 {
		notices = append(notices, fmt.Sprintf(
			"─── ⚠️ CLI version %s is below minimum supported %s — update required",
			current, minV,
		))
	}

	if len(notices) > 0 {
		fmt.Fprintln(os.Stderr)
		for _, n := range notices {
			fmt.Fprintln(os.Stderr, n)
		}
	}
}

// init replaces the placeholder printNotices from output.go with
// the real implementation.
func init() {
	printNotices = printNoticesImpl
}