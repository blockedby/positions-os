// Package main implements the Telegram authentication CLI tool.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/celestix/gotgproto/storage"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
	"github.com/mdp/qrterminal/v3"
)

func main() {
	fmt.Println("=== Telegram QR Auth Tool ===")
	fmt.Println("Generates a session string for Telegram API via QR code")

	// Load .env file if exists
	_ = godotenv.Load()

	// Get API credentials with interactive confirmation
	apiID, apiHash := getAPICredentials()

	// Perform QR authentication
	authWithQR(apiID, apiHash)
}

// getAPICredentials loads from .env with interactive confirmation
func getAPICredentials() (int, string) {
	reader := bufio.NewReader(os.Stdin)

	// Load TG_API_ID
	apiIDStr := os.Getenv("TG_API_ID")
	if apiIDStr != "" {
		fmt.Printf("Found TG_API_ID in .env: %s\n", apiIDStr)
		fmt.Print("Press Y to accept or enter a different value: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" && strings.ToUpper(input) != "Y" {
			apiIDStr = input
		}
	} else {
		fmt.Print("Enter your API ID (from https://my.telegram.org): ")
		apiIDStr, _ = reader.ReadString('\n')
		apiIDStr = strings.TrimSpace(apiIDStr)
	}

	// Load TG_API_HASH
	apiHash := os.Getenv("TG_API_HASH")
	if apiHash != "" {
		fmt.Printf("Found TG_API_HASH in .env: %s\n", maskString(apiHash))
		fmt.Print("Press Y to accept or enter a different value: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" && strings.ToUpper(input) != "Y" {
			apiHash = input
		}
	} else {
		fmt.Print("Enter your API Hash: ")
		apiHash, _ = reader.ReadString('\n')
		apiHash = strings.TrimSpace(apiHash)
	}

	// Parse API ID
	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		fmt.Printf("Error: invalid API ID: %v\n", err)
		os.Exit(1)
	}

	// Debug output
	fmt.Printf("\n[DEBUG] Using API_ID: %d\n", apiID)
	fmt.Printf("[DEBUG] Using API_HASH: %s (length: %d)\n", maskString(apiHash), len(apiHash))
	fmt.Println()

	return apiID, apiHash
}

// maskString masks the middle part of a string for display
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// authWithQR performs QR code authentication with automatic retry on expiration
func authWithQR(apiID int, apiHash string) {
	fmt.Println("Initializing QR login...")

	memStorage := &session.StorageMemory{}
	dispatcher := tg.NewUpdateDispatcher()

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: memStorage,
		UpdateHandler:  dispatcher,
	})

	// Setup graceful shutdown on Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var sessionString string
	var username string

	err := client.Run(ctx, func(ctx context.Context) error {
		qr := client.QR()
		loggedIn := qrlogin.OnLoginToken(dispatcher)

		// Loop to handle QR expiration and regeneration
		for {
			auth, err := qr.Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
				expires := time.Until(token.Expires()).Round(time.Second)

				fmt.Println("╔═══════════════════════════════════════════════════════╗")
				fmt.Println("║  SCAN THIS QR CODE WITH YOUR TELEGRAM APP            ║")
				fmt.Println("║  Settings → Devices → Link Desktop Device            ║")
				fmt.Printf("║  Expires in: %-40s ║\n", expires.String())
				fmt.Println("╚═══════════════════════════════════════════════════════╝")

				qrterminal.GenerateHalfBlock(token.URL(), qrterminal.L, os.Stdout)

				fmt.Printf("\nToken URL: %s\n", token.URL())
				fmt.Println("\nWaiting for scan...")
				return nil
			})

			if err != nil {
				// Check if QR expired
				if strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "timeout") {
					fmt.Println("\n⚠️  QR code expired. Generating a new one...")
					time.Sleep(1 * time.Second)
					continue // Retry with new QR
				}
				return fmt.Errorf("QR auth failed: %w", err)
			}

			// Success - get user info
			user, err := client.Self(ctx)
			if err != nil {
				return fmt.Errorf("failed to get user info: %w", err)
			}

			// Export session string in gotgproto format
			sessionString, err = exportSessionString(ctx, memStorage)
			if err != nil {
				return fmt.Errorf("failed to export session: %w", err)
			}

			if user.Username != "" {
				username = user.Username
			} else {
				username = fmt.Sprintf("%d (%s)", user.ID, user.FirstName)
			}

			_ = auth
			break // Exit loop on success
		}

		return nil
	})

	if err != nil {
		if err == context.Canceled {
			fmt.Println("\n\nAuthentication cancelled by user.")
			os.Exit(0)
		}
		fmt.Printf("\nError during QR login: %v\n", err)
		os.Exit(1)
	}

	printSuccess(username, sessionString)
}

// exportSessionString converts gotd session to gotgproto session string
func exportSessionString(ctx context.Context, memStorage *session.StorageMemory) (string, error) {
	// Get raw session bytes directly (this is what gotgproto does!)
	// LoadSession returns raw bytes that are already properly formatted
	rawSessionBytes, err := memStorage.LoadSession(ctx)
	if err != nil {
		return "", fmt.Errorf("load session: %w", err)
	}

	// Create gotgproto-compatible Session structure
	// Data field should contain the RAW bytes from LoadSession, NOT re-serialized
	gotgSession := storage.Session{
		Version: storage.LatestVersion,
		Data:    rawSessionBytes,
	}

	// Encode to base64 string
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := json.NewEncoder(encoder).Encode(&gotgSession); err != nil {
		return "", fmt.Errorf("encode session: %w", err)
	}
	_ = encoder.Close()

	return buf.String(), nil
}

// printSuccess displays the final result and saves to file
func printSuccess(username, sessionString string) {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║           ✓ AUTHENTICATION SUCCESSFUL!               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Printf("\nLogged in as: @%s\n", username)

	// Save session string to file (avoids PowerShell copy issues)
	sessionFile := "session.txt"
	if err := os.WriteFile(sessionFile, []byte(sessionString), 0600); err != nil {
		fmt.Printf("\n⚠️  Failed to write session file: %v\n", err)
	} else {
		fmt.Printf("\n✅ Session string saved to: %s\n", sessionFile)
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("YOUR SESSION STRING:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(sessionString)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\nCopy from session.txt or paste above into .env as TG_SESSION_STRING")
	fmt.Println("\n⚠️  KEEP THIS SECRET! It provides full access to your account.")
}
