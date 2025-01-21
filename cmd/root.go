package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "OPEN-TODO-PLACEHOLDER",
	Short: "OPEN-TODO-PLACEHOLDER is the social media bot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("No subcommand given")
		cmd.Usage()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Exit with a nonzero exit code if the command fails with an error
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
