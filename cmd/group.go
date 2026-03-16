package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JordanCoin/wom-cli/internal/api"
	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Group info, leaderboards, and competitions",
}

var groupInfoCmd = &cobra.Command{
	Use:     "info [group_id]",
	Short:   "Show group details",
	Args:    cobra.ExactArgs(1),
	Example: `  wom group info 5165`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		data, err := client.GetGroup(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(3)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		name, _ := data["name"].(string)
		memberCount, _ := data["memberCount"].(float64)
		desc, _ := data["description"].(string)
		fmt.Printf("Group: %s\n", name)
		fmt.Printf("Members: %.0f\n", memberCount)
		if desc != "" {
			fmt.Printf("Description: %s\n", desc)
		}
		return nil
	},
}

var groupLeaderboardCmd = &cobra.Command{
	Use:   "leaderboard [group_id]",
	Short: "Show XP/KC gains leaderboard",
	Args:  cobra.ExactArgs(1),
	Example: `  wom group leaderboard 5165 --metric overall --period week --top 10
  wom group leaderboard 5165 --metric runecrafting --period month`,
	RunE: func(cmd *cobra.Command, args []string) error {
		metric, _ := cmd.Flags().GetString("metric")
		period, _ := cmd.Flags().GetString("period")
		top, _ := cmd.Flags().GetInt("top")

		client := api.NewClient()
		results, err := client.GetGroupGains(args[0], metric, period, top)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		fmt.Printf("Top %d %s gains (%s):\n", top, metric, period)
		for i, r := range results {
			if entry, ok := r.(map[string]interface{}); ok {
				player, _ := entry["player"].(map[string]interface{})
				name := "unknown"
				if player != nil {
					name, _ = player["displayName"].(string)
				}
				// Gains are in data.gained subobject
				gained := 0.0
				if data, ok := entry["data"].(map[string]interface{}); ok {
					gained, _ = data["gained"].(float64)
				}
				if gained > 0 {
					// Use decimal format for EHB/EHP metrics, integer for XP/KC
					if metric == "ehb" || metric == "ehp" {
						fmt.Printf("  %d. %s — +%.1f %s\n", i+1, name, gained, metric)
					} else {
						fmt.Printf("  %d. %s — +%s\n", i+1, name, formatNumber(gained))
					}
				}
			}
		}
		return nil
	},
}

var groupHiscoresCmd = &cobra.Command{
	Use:     "hiscores [group_id]",
	Short:   "Show group hiscores for a metric",
	Args:    cobra.ExactArgs(1),
	Example: `  wom group hiscores 5165 --metric overall --top 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		metric, _ := cmd.Flags().GetString("metric")
		top, _ := cmd.Flags().GetInt("top")

		client := api.NewClient()
		results, err := client.GetGroupHiscores(args[0], metric, top)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		fmt.Printf("Hiscores for %s:\n", metric)
		for i, r := range results {
			if entry, ok := r.(map[string]interface{}); ok {
				player, _ := entry["player"].(map[string]interface{})
				name := "unknown"
				if player != nil {
					name, _ = player["displayName"].(string)
				}
				data, _ := entry["data"].(map[string]interface{})
				var value float64
				if data != nil {
					// Could be experience, kills, score depending on metric
					if exp, ok := data["experience"].(float64); ok {
						value = exp
					} else if kills, ok := data["kills"].(float64); ok {
						value = kills
					} else if score, ok := data["score"].(float64); ok {
						value = score
					} else if level, ok := data["level"].(float64); ok {
						value = level
					}
				}
				fmt.Printf("  %d. %s — %s\n", i+1, name, formatNumber(value))
			}
		}
		return nil
	},
}

var groupMembersCmd = &cobra.Command{
	Use:     "members [group_id]",
	Short:   "List all group members",
	Args:    cobra.ExactArgs(1),
	Example: `  wom group members 5165`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		results, err := client.GetGroupMembers(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		fmt.Printf("Members (%d):\n", len(results))
		for _, r := range results {
			if m, ok := r.(map[string]interface{}); ok {
				player, _ := m["player"].(map[string]interface{})
				role, _ := m["role"].(string)
				if player != nil {
					name, _ := player["displayName"].(string)
					fmt.Printf("  %s (%s)\n", name, role)
				}
			}
		}
		return nil
	},
}

var groupCompetitionsCmd = &cobra.Command{
	Use:     "competitions [group_id]",
	Short:   "List group competitions",
	Args:    cobra.ExactArgs(1),
	Example: `  wom group competitions 5165`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		results, err := client.GetGroupCompetitions(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		if len(results) == 0 {
			fmt.Println("No competitions found.")
			return nil
		}
		for _, r := range results {
			if comp, ok := r.(map[string]interface{}); ok {
				title, _ := comp["title"].(string)
				metric, _ := comp["metric"].(string)
				status, _ := comp["status"].(string)
				id, _ := comp["id"].(float64)
				participants, _ := comp["participantCount"].(float64)
				fmt.Printf("  [%.0f] %s (%s) — %s, %.0f participants\n",
					id, title, metric, strings.ToUpper(status), participants)
			}
		}
		return nil
	},
}

func init() {
	groupLeaderboardCmd.Flags().String("metric", "overall", "Metric: overall, attack, slayer, vorkath, etc.")
	groupLeaderboardCmd.Flags().String("period", "week", "Period: day, week, month, year")
	groupLeaderboardCmd.Flags().Int("top", 10, "Number of results")

	groupHiscoresCmd.Flags().String("metric", "overall", "Metric: overall, attack, slayer, etc.")
	groupHiscoresCmd.Flags().Int("top", 10, "Number of results")

	groupCmd.AddCommand(groupInfoCmd)
	groupCmd.AddCommand(groupLeaderboardCmd)
	groupCmd.AddCommand(groupHiscoresCmd)
	groupCmd.AddCommand(groupMembersCmd)
	groupCmd.AddCommand(groupCompetitionsCmd)
}
