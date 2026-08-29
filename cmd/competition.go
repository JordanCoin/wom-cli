package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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
or group-based (with group ID + verification code).

The group ID and the verification code both fall back to $WOM_GROUP_ID and
$WOM_VERIFICATION_CODE when their flags are absent, so a caller never has to
put the group's secret in argv.

A standalone competition mints its own verification code, and this command's
output is the only place it is ever shown, so that one is printed. A code you
supplied is never echoed back: you already have it, and printing it would copy
a long-lived group secret into stdout.`,
	Example: `  wom competition create --title "RC SOTW" --metric runecrafting --starts "2026-03-20T00:00:00Z" --ends "2026-03-27T00:00:00Z" --participants "Zezima,Lynx Titan"
  wom competition create --title "PVM BOTW" --metric vorkath --starts "2026-03-20T00:00:00Z" --ends "2026-03-27T00:00:00Z" --group-id 5165 --verification-code "123-456-789"
  wom competition create --title "Summer Bingo" --metric ehb --starts "2026-06-13T00:00:00Z" --ends "2026-06-27T00:00:00Z" --group-id 5165 --verification-code "123-456-789" --team "Team Alpha=Doe Matic,Uka36" --team "Team Bravo=Zezima"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		metric, _ := cmd.Flags().GetString("metric")
		starts, _ := cmd.Flags().GetString("starts")
		ends, _ := cmd.Flags().GetString("ends")
		participantsStr, _ := cmd.Flags().GetString("participants")
		teamSpecs, _ := cmd.Flags().GetStringArray("team")
		groupIDFlag, _ := cmd.Flags().GetString("group-id")
		verificationCodeFlag, _ := cmd.Flags().GetString("verification-code")
		groupID := resolveSecret(groupIDFlag, envGroupID)
		verificationCode := resolveSecret(verificationCodeFlag, envVerificationCode)

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

		teams, err := parseTeams(teamSpecs)
		if err != nil {
			return err
		}

		client := api.NewClient()
		data, err := client.CreateCompetition(title, metric, starts, ends, participants, teams, groupID, verificationCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		return renderCreateOutput(cmd.OutOrStdout(), data, title, metric, starts, ends, verificationCode, jsonOutput)
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
		codeFlag, _ := cmd.Flags().GetString("verification-code")
		code := resolveSecret(codeFlag, envVerificationCode)
		if code == "" {
			return missingCodeError()
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
		codeFlag, _ := cmd.Flags().GetString("verification-code")
		code := resolveSecret(codeFlag, envVerificationCode)
		if code == "" {
			return missingCodeError()
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
		codeFlag, _ := cmd.Flags().GetString("verification-code")
		code := resolveSecret(codeFlag, envVerificationCode)
		playersStr, _ := cmd.Flags().GetString("players")
		if code == "" {
			return missingCodeError()
		}
		if playersStr == "" {
			return fmt.Errorf("--players is required")
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
		codeFlag, _ := cmd.Flags().GetString("verification-code")
		code := resolveSecret(codeFlag, envVerificationCode)
		playersStr, _ := cmd.Flags().GetString("players")
		if code == "" {
			return missingCodeError()
		}
		if playersStr == "" {
			return fmt.Errorf("--players is required")
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
		codeFlag, _ := cmd.Flags().GetString("verification-code")
		code := resolveSecret(codeFlag, envVerificationCode)
		if code == "" {
			return missingCodeError()
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
	competitionCreateCmd.Flags().String("participants", "", "Comma-separated participant usernames (classic competition)")
	competitionCreateCmd.Flags().StringArray("team", nil, "A team, as 'Name=player1,player2'. Repeat for each team. Makes it a team competition, so it cannot be combined with --participants")
	competitionCreateCmd.Flags().String("group-id", "", groupIDFlagHelp)
	competitionCreateCmd.Flags().String("verification-code", "", verificationCodeFlagHelp)

	competitionEditCmd.Flags().String("title", "", "New title")
	competitionEditCmd.Flags().String("ends", "", "New end time ISO 8601")
	competitionEditCmd.Flags().String("participants", "", "Replace participant list (comma-separated)")
	competitionEditCmd.Flags().String("verification-code", "", verificationCodeFlagHelp)

	competitionAddParticipantsCmd.Flags().String("players", "", "Comma-separated usernames to add (required)")
	competitionAddParticipantsCmd.Flags().String("verification-code", "", verificationCodeFlagHelp)

	competitionRemoveParticipantsCmd.Flags().String("players", "", "Comma-separated usernames to remove (required)")
	competitionRemoveParticipantsCmd.Flags().String("verification-code", "", verificationCodeFlagHelp)

	competitionUpdateAllCmd.Flags().String("verification-code", "", verificationCodeFlagHelp)

	competitionDeleteCmd.Flags().String("verification-code", "", verificationCodeFlagHelp)

	competitionCmd.AddCommand(competitionCreateCmd)
	competitionCmd.AddCommand(competitionViewCmd)
	competitionCmd.AddCommand(competitionEditCmd)
	competitionCmd.AddCommand(competitionAddParticipantsCmd)
	competitionCmd.AddCommand(competitionRemoveParticipantsCmd)
	competitionCmd.AddCommand(competitionUpdateAllCmd)
	competitionCmd.AddCommand(competitionDeleteCmd)

	rootCmd.AddCommand(competitionCmd)
}

// parseTeams turns repeated `--team "Name=player1,player2"` flags into the
// shape the API client wants.
//
// `=` splits the name from the roster and `,` splits the roster, which is
// unambiguous because an OSRS name is at most 12 characters of letters, digits,
// spaces and underscores: it can never contain either separator. Splitting the
// name on the FIRST `=` rather than the last leaves a team free to have one in
// its name, which is likelier than a player having one.
func parseTeams(specs []string) ([]api.TeamSpec, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	teams := make([]api.TeamSpec, 0, len(specs))
	seen := make(map[string]bool, len(specs))

	for _, spec := range specs {
		name, roster, found := strings.Cut(spec, "=")
		if !found {
			return nil, fmt.Errorf("--team %q is missing its players: write it as 'Name=player1,player2'", spec)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("--team %q has no team name before the '='", spec)
		}

		// WOM keys a team by its name, so two teams sharing one would silently
		// become a single team with a merged roster.
		if seen[strings.ToLower(name)] {
			return nil, fmt.Errorf("--team %q is named twice", name)
		}
		seen[strings.ToLower(name)] = true

		players := make([]string, 0, 4)
		for _, p := range strings.Split(roster, ",") {
			if p = strings.TrimSpace(p); p != "" {
				players = append(players, p)
			}
		}
		if len(players) == 0 {
			return nil, fmt.Errorf("--team %q lists no players", name)
		}

		teams = append(teams, api.TeamSpec{Name: name, Participants: players})
	}

	return teams, nil
}

// renderCreateOutput writes what `competition create` says about a
// competition that was just made.
//
// WOM's create response echoes a `verificationCode`. Which of the two codes
// that is decides whether it may be printed:
//
//   - A standalone competition mints its own, and this response is the only
//     place it is ever shown. Swallowing it would lose the only thing that
//     can later edit or delete the competition.
//   - A group competition's response echoes the code the caller just
//     supplied. Printing that copies a long-lived group secret into stdout,
//     and from there into a shell transcript or an agent's tool log, in
//     exchange for telling the caller something they already know.
//
// `suppliedCode` is how the two are told apart: non-empty means the caller
// brought their own. The check covers the `--json` path as well as the
// human one, because the JSON path is the one an agent reads.
func renderCreateOutput(w io.Writer, data map[string]interface{}, title, metric, starts, ends, suppliedCode string, asJSON bool) error {
	if suppliedCode != "" {
		delete(data, "verificationCode")
	}

	if asJSON {
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
		return nil
	}

	if comp, ok := data["competition"].(map[string]interface{}); ok {
		id, _ := comp["id"].(float64)
		fmt.Fprintf(w, "Competition '%s' created! (ID: %.0f)\n", title, id)
		fmt.Fprintf(w, "Metric: %s\n", metric)
		fmt.Fprintf(w, "Starts: %s\n", starts)
		fmt.Fprintf(w, "Ends: %s\n", ends)
	}
	if code, ok := data["verificationCode"].(string); ok {
		fmt.Fprintf(w, "Verification code: %s (save this, it is what edits or deletes the competition)\n", code)
	}
	return nil
}
