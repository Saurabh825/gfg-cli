package cmd

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/Saurabh825/gfg-cli/internal/config"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	langOpt      string
	cookiesOpt   bool
	cookieStrOpt string
	showOpt      bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration using flags.",
	Run: func(cmd *cobra.Command, args []string) {
		hasUpdates := false

		if langOpt != "" {
			validLangs := map[string]bool{"cpp": true, "java": true, "python": true, "javascript": true}
			if validLangs[langOpt] {
				config.Cfg.Language = langOpt
				fmt.Printf("✔ Preferred language set to: %s\n", langOpt)
				hasUpdates = true
			} else {
				fmt.Printf("❌ Invalid language. Choose from: cpp, java, python, javascript\n")
			}
		}

		if cookiesOpt {
			fmt.Println("Enter your GFG authentication details:")
			fmt.Print("Enter your sessionid: ")
			sessionBytes, _ := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			sessionID := strings.TrimSpace(string(sessionBytes))

			fmt.Print("Enter your gfguserName: ")
			userBytes, _ := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			username := strings.TrimSpace(string(userBytes))

			if sessionID != "" {
				config.Cfg.SessionID = sessionID
			}
			if username != "" {
				config.Cfg.GfgUserName = username
			}

			if sessionID != "" || username != "" {
				fmt.Println("✔ Authentication cookies saved successfully!")
				hasUpdates = true
			} else {
				fmt.Println("No changes made to cookies.")
			}
		}

		if cookieStrOpt != "" {
			config.Cfg.CookieString = cookieStrOpt
			fmt.Println("✔ Full raw cookie string saved successfully!")
			hasUpdates = true
		}

		if hasUpdates {
			config.Save()
		}

		if showOpt || !hasUpdates {
			if !showOpt && !hasUpdates {
				fmt.Println("Current Configuration:")
				fmt.Println(strings.Repeat("-", 20))
			}

			fmt.Printf("Config path:   %s\n", config.GetConfigPath())
			fmt.Printf("Language:      %s\n", config.Cfg.Language)

			if config.Cfg.CookieString != "" {
				fmt.Println("Cookie String: ********** (Masked)")
			} else {
				fmt.Println("Cookie String: Not set")
			}

			if config.Cfg.GfgUserName != "" {
				fmt.Println("gfguserName:   ********** (Masked)")
			} else {
				fmt.Println("gfguserName:   Not set")
			}

			if config.Cfg.SessionID != "" {
				fmt.Println("sessionid:     ********** (Masked)")
			} else {
				fmt.Println("sessionid:     Not set")
			}

			if !showOpt && !hasUpdates {
				fmt.Println("\nUse --help to see available flags for updating configuration.")
			}
		}
	},
}

func init() {
	configCmd.Flags().StringVarP(&langOpt, "lang", "l", "", "Set preferred language")
	configCmd.Flags().BoolVarP(&cookiesOpt, "cookies", "c", false, "Set authentication cookies interactively")
	configCmd.Flags().StringVar(&cookieStrOpt, "cookie-string", "", "Paste your full raw cookie string")
	configCmd.Flags().BoolVar(&showOpt, "show", false, "Show current configuration")
	rootCmd.AddCommand(configCmd)
}
