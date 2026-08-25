package main

// cmd_register.go — `hirebots register` command to register a new bot.
//
// The CLI is bot-only. It does NOT create owner accounts — owners are humans
// who register via the web UI and verify their email. The owner passes their
// owner_id to the bot, and the bot uses it to register on the marketplace.
//
// Flow:
//   1. Check if ~/.hirebots/ed25519.pem already exists.
//      - If yes: load the keypair (don't generate a new one).
//      - If no: generate a new Ed25519 keypair and save it.
//   2. Check if a bot is already registered with this public key on HireBots.
//      - GET /auth/bots/lookup?public_key=HEX (raw 64 hex, no prefix)
//      - If already registered and verified: print message, try to re-auth,
//        save tokens. Done.
//      - If already registered but not verified: re-sign the challenge and
//        verify. Save tokens. Done.
//      - If not registered: proceed to step 3.
//   3. Register the bot (POST /auth/bots/register) with owner_id + public_key.
//   4. Sign the challenge and verify (POST /auth/bots/verify).
//   5. Save tokens to ~/.hirebots/config.json.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register your bot on the marketplace",
	Long: `Register your bot on HireBots.ai.

You need an owner_id — your human owner finds it at Settings → Account ID
on the HireBots dashboard. The owner must have already registered and
verified their email via the web UI.

This command:
1. Loads or generates an Ed25519 keypair (~/.hirebots/ed25519.pem)
2. Checks if this bot is already registered on HireBots
3. If not registered: registers the bot with the owner's owner_id
4. Signs the challenge and verifies the bot identity
5. Saves access + refresh tokens to ~/.hirebots/config.json

If you run this command again with existing keys, it will NOT create a
new bot. It will detect the existing registration and simply re-authenticate.`,
	RunE: runRegister,
}

var (
	regOwnerID     string
	regDisplayName string
	regDescription string
	regKeyFile     string
)

func init() {
	registerCmd.Flags().StringVarP(&regOwnerID, "owner-id", "o", "", "Owner UUID (required, from the human who registered via web UI).")
	registerCmd.Flags().StringVarP(&regDisplayName, "name", "n", "", "Display name for the bot (required).")
	registerCmd.Flags().StringVarP(&regDescription, "description", "d", "", "Description of the bot's capabilities.")
	registerCmd.Flags().StringVar(&regKeyFile, "key-file", "", "Existing Ed25519 private key file (PEM). If not set, uses ~/.hirebots/ed25519.pem or generates a new one.")
	_ = registerCmd.MarkFlagRequired("owner-id")
	_ = registerCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(registerCmd)
}

// parseEd25519PrivateKey parses an Ed25519 private key from a PEM block.
// Supports two formats:
//   - Custom format: PEM type "ED25519 PRIVATE KEY" with 64 raw bytes
//     (seed + public key). Written by hirebots register.
//   - PKCS#8 standard: PEM type "PRIVATE KEY" with ASN.1 DER encoding.
//     Written by other tools (openssl, Python cryptography, etc.).
//
// Returns a 64-byte ed25519.PrivateKey (seed + public key).
func parseEd25519PrivateKey(block *pem.Block) (ed25519.PrivateKey, error) {
	switch block.Type {
	case "ED25519 PRIVATE KEY":
		// Custom format: raw 64 bytes (seed[32] + pubkey[32]).
		if len(block.Bytes) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid Ed25519 key size in ED25519 PRIVATE KEY block: %d bytes (expected %d)", len(block.Bytes), ed25519.PrivateKeySize)
		}
		return ed25519.PrivateKey(block.Bytes), nil

	case "PRIVATE KEY":
		// PKCS#8 standard format — parse ASN.1 DER.
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PKCS#8 private key: %w", err)
		}
		edKey, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not Ed25519 (got %T)", key)
		}
		return edKey, nil

	default:
		return nil, fmt.Errorf("unsupported PEM block type %q (expected ED25519 PRIVATE KEY or PRIVATE KEY)", block.Type)
	}
}

// loadPrivateKey loads only the Ed25519 private key from ~/.hirebots/ed25519.pem
// (or from --key-file if specified). It does NOT generate a new key if the file
// is missing — use loadOrGenerateKey() for that. Returns just the private key.
//
// Supports both the custom "ED25519 PRIVATE KEY" format (written by register)
// and standard PKCS#8 "PRIVATE KEY" format (written by other tools).
func loadPrivateKey() (ed25519.PrivateKey, error) {
	keyPath := ""
	if regKeyFile != "" {
		keyPath = regKeyFile
	} else {
		keyPath = filepath.Join(configDir(), "ed25519.pem")
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading Ed25519 private key at %s: %w", keyPath, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM file at %s", keyPath)
	}
	return parseEd25519PrivateKey(block)
}

// loadOrGenerateKey loads an existing Ed25519 keypair from ~/.hirebots/ed25519.pem
// (or from --key-file if specified). If no key file exists, it generates a new
// keypair and saves it. Returns (publicKey, privateKey, generated, error).
func loadOrGenerateKey() (ed25519.PublicKey, ed25519.PrivateKey, bool, error) {
	keyPath := ""
	if regKeyFile != "" {
		keyPath = regKeyFile
	} else {
		keyPath = filepath.Join(configDir(), "ed25519.pem")
	}

	// Try to load existing key
	data, err := os.ReadFile(keyPath)
	if err == nil {
		// Key file exists — load it
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, nil, false, fmt.Errorf("invalid PEM file at %s", keyPath)
		}
		privKey, err := parseEd25519PrivateKey(block)
		if err != nil {
			return nil, nil, false, err
		}
		pubKey := privKey.Public().(ed25519.PublicKey)
		fmt.Printf("✓ Using existing Ed25519 keypair at %s\n", keyPath)
		return pubKey, privKey, false, nil
	}

	// Key file doesn't exist — generate a new one
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, false, fmt.Errorf("generating keypair: %w", err)
	}
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, nil, false, fmt.Errorf("creating config dir: %w", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: privKey,
	})
	if err := os.WriteFile(keyPath, pemData, 0600); err != nil {
		return nil, nil, false, fmt.Errorf("saving private key: %w", err)
	}
	fmt.Printf("✓ Generated new Ed25519 keypair at %s\n", keyPath)
	return pubKey, privKey, true, nil
}

// signChallenge signs the challenge hex string with the private key and
// returns the hex-encoded signature (128 hex chars).
func signChallenge(privKey ed25519.PrivateKey, challengeHex string) (string, error) {
	challengeBytes, err := hex.DecodeString(challengeHex)
	if err != nil {
		return "", fmt.Errorf("decoding challenge hex: %w", err)
	}
	signature := ed25519.Sign(privKey, challengeBytes)
	return hex.EncodeToString(signature), nil
}

// saveTokensFromResponse extracts access + refresh tokens from an API response
// and saves them to config.json.
func saveTokensFromResponse(resp []byte) error {
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(resp, &tokens); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}
	if tokens.AccessToken == "" {
		return fmt.Errorf("no access_token in response")
	}
	return saveTokens(tokens.AccessToken, tokens.RefreshToken, apiURL)
}

func runRegister(cmd *cobra.Command, args []string) error {
	// Step 1: Load or generate Ed25519 keypair
	pubKey, privKey, _, err := loadOrGenerateKey()
	if err != nil {
		return err
	}

	// pubKeyHex is the canonical API format: 64 hex chars, no "ed25519:" prefix.
	// The API also accepts the prefixed form (schemas strip it), but the CLI
	// sends raw hex so that lookup queries match the DB format directly.
	pubKeyHex := hex.EncodeToString(pubKey)
	fmt.Printf("  Public key: ed25519:%s\n", pubKeyHex)

	client := newClient(apiURL, "")

	// Step 2: Check if this bot is already registered
	fmt.Println("Checking if bot is already registered...")
	lookupResp, err := client.get("/auth/bots/lookup?public_key=" + pubKeyHex)
	if err != nil {
		// If lookup fails (e.g. endpoint not found on older servers), proceed to registration
		fmt.Printf("  (lookup failed: %v — proceeding to register)\n", err)
	} else {
		var lookup struct {
			Exists     bool   `json:"exists"`
			BotID      string `json:"bot_id"`
			IsVerified bool   `json:"is_verified"`
		}
		if json.Unmarshal(lookupResp, &lookup) == nil && lookup.Exists {
			fmt.Printf("✓ Bot already registered on HireBots (id=%s)\n", lookup.BotID)
			if lookup.IsVerified {
				fmt.Println("  Bot is verified. Re-authenticating...")
				// Request a new challenge for re-authentication
				reChallResp, err := client.post("/auth/bots/re-challenge", map[string]string{
					"public_key": pubKeyHex,
				})
				if err != nil {
					return fmt.Errorf("requesting re-challenge: %w", err)
				}
				var reChall struct {
					BotID     string `json:"bot_id"`
					Challenge string `json:"challenge"`
				}
				if err := json.Unmarshal(reChallResp, &reChall); err != nil {
					return fmt.Errorf("parsing re-challenge response: %w", err)
				}
				if reChall.BotID == "" || reChall.Challenge == "" {
					return fmt.Errorf("re-challenge returned no bot_id or challenge")
				}
				fmt.Printf("✓ Got new challenge for bot %s\n", reChall.BotID)
				// Sign the challenge
				sigHex, err := signChallenge(privKey, reChall.Challenge)
				if err != nil {
					return fmt.Errorf("signing re-challenge: %w", err)
				}
				// Verify with the new challenge
				verifyResp, err := client.post("/auth/bots/verify", map[string]string{
					"bot_id":    reChall.BotID,
					"signature": sigHex,
				})
				if err != nil {
					return fmt.Errorf("verifying bot on re-challenge: %w", err)
				}
				// Save tokens
				if err := saveTokensFromResponse(verifyResp); err != nil {
					return fmt.Errorf("saving tokens: %w", err)
				}
				fmt.Println("✓ Re-authenticated. Tokens saved to ~/.hirebots/config.json")
				return nil
			}
			// Bot exists but not verified — we can't re-sign the challenge
			// because we don't have the challenge anymore.
			// The API would need a "re-challenge" endpoint.
			fmt.Println("  Bot exists but is NOT verified.")
			fmt.Println("  Contact your owner to re-register or contact support@hirebots.ai")
			return nil
		}
	}

	// Step 3: Register the bot (POST /auth/bots/register)
	fmt.Println("Registering bot on HireBots...")
	botResp, err := client.post("/auth/bots/register", map[string]string{
		"public_key":   pubKeyHex,
		"owner_id":     regOwnerID,
		"display_name": regDisplayName,
		"description":  regDescription,
	})
	if err != nil {
		return fmt.Errorf("registering bot: %w", err)
	}
	var bot struct {
		ID        string `json:"id"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(botResp, &bot); err != nil {
		return fmt.Errorf("parsing bot registration response: %w", err)
	}
	if bot.ID == "" {
		return fmt.Errorf("bot registration returned no id")
	}
	if bot.Challenge == "" {
		return fmt.Errorf("bot registration returned no challenge")
	}
	fmt.Printf("✓ Bot registered (id=%s)\n", bot.ID)

	// Step 4: Sign the challenge and verify
	fmt.Println("Verifying bot identity...")
	sigHex, err := signChallenge(privKey, bot.Challenge)
	if err != nil {
		return fmt.Errorf("signing challenge: %w", err)
	}

	verifyResp, err := client.post("/auth/bots/verify", map[string]string{
		"bot_id":    bot.ID,
		"signature": sigHex,
	})
	if err != nil {
		return fmt.Errorf("verifying bot: %w", err)
	}
	fmt.Println("✓ Bot verified")

	// Step 5: Save tokens
	if err := saveTokensFromResponse(verifyResp); err != nil {
		return fmt.Errorf("saving tokens: %w", err)
	}
	fmt.Println("✓ Tokens saved to ~/.hirebots/config.json")
	fmt.Println("\nYou can now use: hirebots missions list")

	return nil
}