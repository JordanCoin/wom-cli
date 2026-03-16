package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JordanCoin/wom-cli/internal/api"
	"github.com/spf13/cobra"
)

var competitionCmd = &cobra.Command{
	Use:   "competition",
	Short: "Create and manage WOM competitions",
}

var competitionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new competition",
	Long: `Create a competition on WiseOldMan. Can be standalone (with participants list)
or group-based (with group ID + verification code).`,
	Example: `  wom competition create --title "RC SOTW" --metric runecrafting --starts "2026-03-20T00:00:00Z" --ends "2026-03-27T00:00:00Z" --participants "Zezima,Lynx Titan"
  wom competition create --title "PVM BOTW" --metric vorkath --starts "2026-03-20T00:00:00Z" --ends "2026-03-27T00:00:00Z" --group-id 5165 --verification-code "123-456-789"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		metric, _ := cmd.Flags().GetString("metric")
		starts, _ := cmd.Flags().GetString("starts")
		ends, _ := cmd.Flags().GetString("ends")
		participantsStr, _ := cmd.Flags().GetString("participants")
		groupID, _ := cmd.Flags().GetString("group-id")
		verificationCode, _ := cmd.Flags().GetString("verification-code")

		if title == "" || metric == "" || starts == "" || ends == "" {
			return fmt.Errorf("--title, --metric, --starts, and --ends are required")
		}

		var participants []string
		if participantsStr != "" {
			participants = strings.Split(participantsStr, ",")
			for i := range participants {
				participants[i] = strings.TrimSpace(participants[i])
			}
		}

		client := api.NewClient()
		data, err := client.CreateCompetition(title, metric, starts, ends, participants, groupID, verificationCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		if comp, ok := data["competition"].(map[string]interface{}); ok {
			id, _ := comp["id"].(float64)
			fmt.Printf("Competition '%s' created! (ID: %.0f)\n", title, id)
			fmt.Printf("Metric: %s\n", metric)
			fmt.Printf("Starts: %s\n", starts)
			fmt.Printf("Ends: %s\n", ends)
		}
		if code, ok := data["verificationCode"].(string); ok {
			fmt.Printf("Verification code: %s (save this — needed to edit/delete)\n", code)
		}
		return nil
	},
}

var competitionViewCmd = &cobra.Command{
	Use:     "view [competition_id]",
	Short:   "View competition details and standings",
	Args:    cobra.ExactArgs(1),
	Example: `  wom competition view 12345`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		data, err := client.GetCompetition(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(3)
		}

		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		title, _ := data["title"].(string)
		metric, _ := data["metric"].(string)
		status, _ := data["status"].(string)
		participantCount, _ := data["participantCount"].(float64)

		fmt.Printf("Competition: %s\n", title)
		fmt.Printf("Metric: %s | Status: %s | Participants: %.0f\n", metric, strings.ToUpper(status), participantCount)

		if participations, ok := data["participations"].([]interface{}); ok {
			fmt.Println("\nStandings:")
			for i, p := range participations {
				if i >= 10 {
					fmt.Printf("  ... and %.0f more\n", participantCount-10)
					break
				}
				if entry, ok := p.(map[string]interface{}); ok {
					player, _ := entry["player"].(map[string]interface{})
					name := "unknown"
					if player != nil {
						name, _ = player["displayName"].(string)
					}
					progress, _ := entry["progress"].(map[string]interface{})
					gained := 0.0
					if progress != nil {
						gained, _ = progress["gained"].(float64)
					}
					fmt.Printf("  %d. %s — +%s\n", i+1, name, formatNumber(gained))
				}
			}
		}
		return nil
	},
}

var competitionDeleteCmd = &cobra.Command{
	Use:     "delete [competition_id]",
	Short:   "Delete a competition",
	Args:    cobra.ExactArgs(1),
	Example: `  wom competition delete 12345 --verification-code "123-456-789"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		code, _ := cmd.Flags().GetString("verification-code")
		if code == "" {
			return fmt.Errorf("--verification-code is required to delete a competition")
		}

		client := api.NewClient()
		err := client.DeleteCompetition(args[0], code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			out, _ := json.Marshal(map[string]interface{}{"action": "deleted", "competition_id": args[0]})
			fmt.Println(string(out))
		} else {
			fmt.Printf("Competition %s deleted.\n", args[0])
		}
		return nil
	},
}

var competitionEditCmd = &cobra.Command{
	Use:   "edit [competition_id]",
	Short: "Edit a competition (title, end date, participants)",
	Args:  cobra.ExactArgs(1),
	Example: `  wom competition edit 12345 --title "RC SOTW Week 2" --verification-code "123-456-789"
  wom competition edit 12345 --ends "2026-03-30T00:00:00Z" --verification-code "123-456-789"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		code, _ := cmd.Flags().GetString("verification-code")
		if code == "" {
			return fmt.Errorf("--verification-code is required")
		}
		title, _ := cmd.Flags().GetString("title")
		ends, _ := cmd.Flags().GetString("ends")
		participantsStr, _ := cmd.Flags().GetString("participants")

		body := map[string]interface{}{
			"verificationCode": code,
		}
		if title != "" {
			body["title"] = title
		}
		if ends != "" {
			body["endsAt"] = ends
		}
		if participantsStr != "" {
			parts := strings.Split(participantsStr, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			body["participants"] = parts
		}

		client := api.NewClient()
		data, err := client.EditCompetition(args[0], body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("Competition %s updated.\n", args[0])
		}
		return nil
	},
}

var competitionAddParticipantsCmd = &cobra.Command{
	Use:     "add-participants [competition_id]",
	Short:   "Add participants to a competition",
	Args:    cobra.ExactArgs(1),
	Example: `  wom competition add-participants 12345 --players "Doe Matic,Uka36" --verification-code "123-456-789"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		code, _ := cmd.Flags().GetString("verification-code")
		playersStr, _ := cmd.Flags().GetString("players")
		if code == "" || playersStr == "" {
			return fmt.Errorf("--verification-code and --players are required")
		}
		players := strings.Split(playersStr, ",")
		for i := range players {
			players[i] = strings.TrimSpace(players[i])
		}

		client := api.NewClient()
		err := client.AddParticipants(args[0], players, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.Marshal(map[string]interface{}{"action": "participants_added", "players": players})
			fmt.Println(string(out))
		} else {
			fmt.Printf("Added %d participants to competition %s\n", len(players), args[0])
		}
		return nil
	},
}

var competitionRemoveParticipantsCmd = &cobra.Command{
	Use:     "remove-participants [competition_id]",
	Short:   "Remove participants from a competition",
	Args:    cobra.ExactArgs(1),
	Example: `  wom competition remove-participants 12345 --players "BadPlayer" --verification-code "123-456-789"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		code, _ := cmd.Flags().GetString("verification-code")
		playersStr, _ := cmd.Flags().GetString("players")
		if code == "" || playersStr == "" {
			return fmt.Errorf("--verification-code and --players are required")
		}
		players := strings.Split(playersStr, ",")
		for i := range players {
			players[i] = strings.TrimSpace(players[i])
		}

		client := api.NewClient()
		err := client.RemoveParticipants(args[0], players, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.Marshal(map[string]interface{}{"action": "participants_removed", "players": players})
			fmt.Println(string(out))
		} else {
			fmt.Printf("Removed %d participants from competition %s\n", len(players), args[0])
		}
		return nil
	},
}

var competitionUpdateAllCmd = &cobra.Command{
	Use:     "update-all [competition_id]",
	Short:   "Refresh all outdated participants' stats",
	Args:    cobra.ExactArgs(1),
	Example: `  wom competition update-all 12345 --verification-code "123-456-789"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		code, _ := cmd.Flags().GetString("verification-code")
		if code == "" {
			return fmt.Errorf("--verification-code is required")
		}

		client := api.NewClient()
		err := client.UpdateAllParticipants(args[0], code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.Marshal(map[string]interface{}{"action": "update_all_queued", "competition_id": args[0]})
			fmt.Println(string(out))
		} else {
			fmt.Printf("Queued stat refresh for all participants in competition %s\n", args[0])
		}
		return nil
	},
}

func init() {
	competitionCreateCmd.Flags().String("title", "", "Competition title (required)")
	competitionCreateCmd.Flags().String("metric", "", "Metric: runecrafting, vorkath, overall, etc. (required)")
	competitionCreateCmd.Flags().String("starts", "", "Start time ISO 8601 (required)")
	competitionCreateCmd.Flags().String("ends", "", "End time ISO 8601 (required)")
	competitionCreateCmd.Flags().String("participants", "", "Comma-separated participant usernames")
	competitionCreateCmd.Flags().String("group-id", "", "Group ID (for group competitions)")
	competitionCreateCmd.Flags().String("verification-code", "", "Group verification code (for group competitions)")

	competitionEditCmd.Flags().String("title", "", "New title")
	competitionEditCmd.Flags().String("ends", "", "New end time ISO 8601")
	competitionEditCmd.Flags().String("participants", "", "Replace participant list (comma-separated)")
	competitionEditCmd.Flags().String("verification-code", "", "Verification code (required)")

	competitionAddParticipantsCmd.Flags().String("players", "", "Comma-separated usernames to add (required)")
	competitionAddParticipantsCmd.Flags().String("verification-code", "", "Verification code (required)")

	competitionRemoveParticipantsCmd.Flags().String("players", "", "Comma-separated usernames to remove (required)")
	competitionRemoveParticipantsCmd.Flags().String("verification-code", "", "Verification code (required)")

	competitionUpdateAllCmd.Flags().String("verification-code", "", "Verification code (required)")

	competitionDeleteCmd.Flags().String("verification-code", "", "Verification code (required)")

	competitionCmd.AddCommand(competitionCreateCmd)
	competitionCmd.AddCommand(competitionViewCmd)
	competitionCmd.AddCommand(competitionEditCmd)
	competitionCmd.AddCommand(competitionAddParticipantsCmd)
	competitionCmd.AddCommand(competitionRemoveParticipantsCmd)
	competitionCmd.AddCommand(competitionUpdateAllCmd)
	competitionCmd.AddCommand(competitionDeleteCmd)

	rootCmd.AddCommand(competitionCmd)
}
