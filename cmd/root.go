package cmd

import (
	"github.com/spf13/cobra"
)

var jsonOutput bool
var appVersion string

func SetVersion(v string) { appVersion = v }

var rootCmd = &cobra.Command{
	Use:   "wom",
	Short: "WiseOldMan CLI — OSRS player stats, gains, and leaderboards",
	Long: `Query the WiseOldMan API for Old School RuneScape player stats,
XP gains, group leaderboards, and competitions.

Agent-friendly: supports --json output, meaningful exit codes, no interactive prompts.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(playerCmd)
	rootCmd.AddCommand(groupCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("wom " + appVersion)
		},
	})
}
