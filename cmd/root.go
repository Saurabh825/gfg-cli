package cmd

import (
	"fmt"
	"os"

	"github.com/Saurabh825/gfg-cli/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "gfg",
	Short:   "GFG CLI - Solve GeeksforGeeks problems from your terminal.",
	Version: "dev",
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&config.Debug, "debug", "d", false, "Enable debug mode")
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
