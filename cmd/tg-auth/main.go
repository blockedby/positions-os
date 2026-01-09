package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/gotd/td/session/tdesktop"
)

func main() {
	fmt.Println("=== telegram auth tool ===")
	fmt.Println("this tool generates a session string for telegram api")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// try to detect telegram desktop
	tdataPath := getTelegramDesktopPath()
	accounts, tdataErr := tdesktop.Read(tdataPath, nil)

	// if default path failed, try asking user
	if tdataErr != nil || len(accounts) == 0 {
		fmt.Printf("default path not found: %s\n", tdataPath)
		fmt.Print("enter telegram desktop path (or press enter to skip): ")
		customPath, _ := reader.ReadString('\n')
		customPath = strings.TrimSpace(customPath)

		if customPath != "" {
			// add tdata subfolder if not present
			if !strings.HasSuffix(customPath, "tdata") {
				customPath = filepath.Join(customPath, "tdata")
			}
			accounts, tdataErr = tdesktop.Read(customPath, nil)
			if tdataErr == nil && len(accounts) > 0 {
				tdataPath = customPath
			}
		}
	}

	var authMethod int

	if tdataErr == nil && len(accounts) > 0 {
		fmt.Printf("\ndetected %d telegram desktop session(s) at: %s\n", len(accounts), tdataPath)
		fmt.Println()
		fmt.Println("choose authentication method:")
		fmt.Println("  1. use telegram desktop session (recommended)")
		fmt.Println("  2. authenticate with phone number (sms/code)")
		fmt.Print("\nenter choice [1]: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "2" {
			authMethod = 2
		} else {
			authMethod = 1
		}
	} else {
		fmt.Println("no telegram desktop session found, using phone auth")
		authMethod = 2
	}

	// get api credentials
	apiID, apiHash := getAPICredentials(reader)

	var client *gotgproto.Client
	var err error

	if authMethod == 1 {
		client, err = authWithTData(apiID, apiHash, accounts, reader)
	} else {
		client, err = authWithPhone(apiID, apiHash, reader)
	}

	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer client.Stop()

	// export session string
	sessionString, err := client.ExportStringSession()
	if err != nil {
		fmt.Printf("error exporting session: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ authentication successful!")
	fmt.Printf("logged in as: @%s\n", client.Self.Username)
	fmt.Println("\nyour session string:")
	fmt.Println("---")
	fmt.Println(sessionString)
	fmt.Println("---")
	fmt.Println("\nadd this to your .env file as TG_SESSION_STRING")
	fmt.Println("\n⚠️  keep this secret! it provides full access to your telegram account")
}

// getTelegramDesktopPath returns the path to Telegram Desktop data directory
func getTelegramDesktopPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Telegram Desktop", "tdata")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Telegram Desktop", "tdata")
	default: // linux
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "TelegramDesktop", "tdata")
	}
}

// getAPICredentials reads API ID and Hash from env or prompts user
func getAPICredentials(reader *bufio.Reader) (int, string) {
	apiIDStr := os.Getenv("TG_API_ID")
	apiHash := os.Getenv("TG_API_HASH")

	if apiIDStr == "" {
		fmt.Print("enter your api_id (from https://my.telegram.org): ")
		apiIDStr, _ = reader.ReadString('\n')
		apiIDStr = strings.TrimSpace(apiIDStr)
	}
	if apiHash == "" {
		fmt.Print("enter your api_hash: ")
		apiHash, _ = reader.ReadString('\n')
		apiHash = strings.TrimSpace(apiHash)
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		fmt.Printf("error: invalid api_id: %v\n", err)
		os.Exit(1)
	}

	return apiID, apiHash
}

// authWithTData authenticates using Telegram Desktop session
func authWithTData(apiID int, apiHash string, accounts []tdesktop.Account, reader *bufio.Reader) (*gotgproto.Client, error) {
	var selectedAccount tdesktop.Account

	if len(accounts) == 1 {
		selectedAccount = accounts[0]
		fmt.Println("\nusing the only available account")
	} else {
		fmt.Printf("\nfound %d telegram accounts:\n", len(accounts))
		for i, acc := range accounts {
			// try to get some info about the account
			fmt.Printf("  %d. Account #%d\n", i+1, i+1)
			_ = acc // account doesn't expose user info directly
		}

		fmt.Print("\nselect account number [1]: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		idx := 0
		if choice != "" {
			n, err := strconv.Atoi(choice)
			if err == nil && n >= 1 && n <= len(accounts) {
				idx = n - 1
			}
		}
		selectedAccount = accounts[idx]
	}

	fmt.Println("\nauthenticating with telegram desktop session...")

	client, err := gotgproto.NewClient(
		apiID,
		apiHash,
		gotgproto.ClientTypePhone(""), // empty = use session
		&gotgproto.ClientOpts{
			Session:          sessionMaker.TdataSession(selectedAccount).Name("tdata_session"),
			DisableCopyright: true,
		},
	)

	return client, err
}

// authWithPhone authenticates using phone number (SMS/code)
func authWithPhone(apiID int, apiHash string, reader *bufio.Reader) (*gotgproto.Client, error) {
	fmt.Print("enter your phone number (with country code, e.g. +1234567890): ")
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)

	fmt.Println("\nauthenticating... (check telegram for code)")

	client, err := gotgproto.NewClient(
		apiID,
		apiHash,
		gotgproto.ClientTypePhone(phone),
		&gotgproto.ClientOpts{
			Session:          sessionMaker.SqlSession(sqlite.Open("tg_session")),
			DisableCopyright: true,
		},
	)

	if err == nil {
		fmt.Println("\nnote: tg_session.db was created for temporary storage.")
		fmt.Println("you can delete it after copying the session string.")
	}

	return client, err
}
