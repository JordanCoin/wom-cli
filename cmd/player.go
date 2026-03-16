package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JordanCoin/wom-cli/internal/api"
	"github.com/spf13/cobra"
)

var playerCmd = &cobra.Command{
	Use:   "player",
	Short: "Look up player stats, gains, achievements, and records",
}

var playerLookupCmd = &cobra.Command{
	Use:     "lookup [username]",
	Short:   "Look up a player's stats",
	Args:    cobra.ExactArgs(1),
	Example: `  wom player lookup Zezima`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		data, err := client.GetPlayer(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(3)
		}

		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		displayName, _ := data["displayName"].(string)
		playerType, _ := data["type"].(string)
		combatLevel, _ := data["combatLevel"].(float64)
		exp, _ := data["exp"].(float64)
		ehp, _ := data["ehp"].(float64)
		ehb, _ := data["ehb"].(float64)

		fmt.Printf("Player: %s (%s)\n", displayName, playerType)
		fmt.Printf("Combat: %.0f\n", combatLevel)
		fmt.Printf("Total XP: %s\n", formatNumber(exp))
		fmt.Printf("EHP: %.1f | EHB: %.1f\n", ehp, ehb)

		if latestSnapshot, ok := data["latestSnapshot"].(map[string]interface{}); ok {
			if skillData, ok := latestSnapshot["data"].(map[string]interface{}); ok {
				if skills, ok := skillData["skills"].(map[string]interface{}); ok {
					if overall, ok := skills["overall"].(map[string]interface{}); ok {
						level, _ := overall["level"].(float64)
						fmt.Printf("Total Level: %.0f\n", level)
					}
				}
				// Show top boss KCs
				if bosses, ok := skillData["bosses"].(map[string]interface{}); ok {
					type bossKC struct {
						name  string
						kills float64
					}
					var topBosses []bossKC
					for name, v := range bosses {
						if b, ok := v.(map[string]interface{}); ok {
							kills, _ := b["kills"].(float64)
							if kills > 0 {
								topBosses = append(topBosses, bossKC{name, kills})
							}
						}
					}
					// Sort by kills desc
					for i := 0; i < len(topBosses); i++ {
						for j := i + 1; j < len(topBosses); j++ {
							if topBosses[j].kills > topBosses[i].kills {
								topBosses[i], topBosses[j] = topBosses[j], topBosses[i]
							}
						}
					}
					if len(topBosses) > 0 {
						fmt.Println("\nTop Bosses:")
						for i, b := range topBosses {
							if i >= 10 {
								break
							}
							fmt.Printf("  %s: %.0f KC\n", formatBossName(b.name), b.kills)
						}
					}
				}
			}
		}
		return nil
	},
}

var playerGainsCmd = &cobra.Command{
	Use:   "gains [username]",
	Short: "Show XP and boss KC gains for a player",
	Args:  cobra.ExactArgs(1),
	Example: `  wom player gains Zezima --period week
  wom player gains 'Doe Matic' --period week --metric vorkath
  wom player gains 'Doe Matic' --metric ranged`,
	RunE: func(cmd *cobra.Command, args []string) error {
		period, _ := cmd.Flags().GetString("period")
		metric, _ := cmd.Flags().GetString("metric")
		client := api.NewClient()
		data, err := client.GetPlayerGains(args[0], period)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(3)
		}

		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		fmt.Printf("Gains for %s (%s):\n", args[0], period)
		if gainsData, ok := data["data"].(map[string]interface{}); ok {
			// Normalize metric for matching (lowercase, underscores)
			metricNorm := strings.ToLower(strings.ReplaceAll(metric, " ", "_"))

			showSkills := metric == "" // show all if no filter
			showBosses := metric == ""

			// Check if metric matches a specific skill or boss
			if metric != "" {
				if skills, ok := gainsData["skills"].(map[string]interface{}); ok {
					if _, ok := skills[metricNorm]; ok {
						showSkills = true
					}
				}
				if bosses, ok := gainsData["bosses"].(map[string]interface{}); ok {
					if _, ok := bosses[metricNorm]; ok {
						showBosses = true
					}
				}
				// If metric didn't match anything, show both and let it filter below
				if !showSkills && !showBosses {
					showSkills = true
					showBosses = true
				}
			}

			// Skill XP gains
			if showSkills {
				if skills, ok := gainsData["skills"].(map[string]interface{}); ok {
					type skillGain struct {
						name string
						xp   float64
					}
					var gains []skillGain
					for name, v := range skills {
						if metric != "" && name != metricNorm {
							continue
						}
						if skill, ok := v.(map[string]interface{}); ok {
							if gained, ok := skill["experience"].(map[string]interface{}); ok {
								if xp, ok := gained["gained"].(float64); ok && xp > 0 {
									gains = append(gains, skillGain{name, xp})
								}
							}
						}
					}
					for i := 0; i < len(gains); i++ {
						for j := i + 1; j < len(gains); j++ {
							if gains[j].xp > gains[i].xp {
								gains[i], gains[j] = gains[j], gains[i]
							}
						}
					}
					if len(gains) > 0 {
						fmt.Println("\nSkill XP Gains:")
						for i, g := range gains {
							if i >= 10 && metric == "" {
								break
							}
							fmt.Printf("  %s: +%s XP\n", g.name, formatNumber(g.xp))
						}
					}
				}
			}

			// Boss KC gains
			if showBosses {
				if bosses, ok := gainsData["bosses"].(map[string]interface{}); ok {
					type bossGain struct {
						name  string
						kills float64
					}
					var bossGains []bossGain
					for name, v := range bosses {
						if metric != "" && name != metricNorm {
							continue
						}
						if b, ok := v.(map[string]interface{}); ok {
							if killData, ok := b["kills"].(map[string]interface{}); ok {
								if gained, ok := killData["gained"].(float64); ok && gained > 0 {
									bossGains = append(bossGains, bossGain{name, gained})
								}
							}
						}
					}
					for i := 0; i < len(bossGains); i++ {
						for j := i + 1; j < len(bossGains); j++ {
							if bossGains[j].kills > bossGains[i].kills {
								bossGains[i], bossGains[j] = bossGains[j], bossGains[i]
							}
						}
					}
					if len(bossGains) > 0 {
						fmt.Println("\nBoss KC Gains:")
						for i, b := range bossGains {
							if i >= 10 && metric == "" {
								break
							}
							fmt.Printf("  %s: +%.0f KC\n", formatBossName(b.name), b.kills)
						}
					}
				}
			}

			// If filtering by metric and nothing was printed, show explicit zero
			if metric != "" {
				foundMatch := false
				if skills, ok := gainsData["skills"].(map[string]interface{}); ok {
					if _, ok := skills[metricNorm]; ok {
						foundMatch = true
					}
				}
				if bosses, ok := gainsData["bosses"].(map[string]interface{}); ok {
					if _, ok := bosses[metricNorm]; ok {
						foundMatch = true
					}
				}
				if foundMatch {
					// Metric exists in API but 0 gains — show explicit zero
					// (output was already printed if gains > 0)
					// Check if we already printed something by looking at bosses/skills
					printed := false
					if skills, ok := gainsData["skills"].(map[string]interface{}); ok {
						if s, ok := skills[metricNorm].(map[string]interface{}); ok {
							if exp, ok := s["experience"].(map[string]interface{}); ok {
								if gained, ok := exp["gained"].(float64); ok && gained > 0 {
									printed = true
								}
							}
						}
					}
					if bosses, ok := gainsData["bosses"].(map[string]interface{}); ok {
						if b, ok := bosses[metricNorm].(map[string]interface{}); ok {
							if kills, ok := b["kills"].(map[string]interface{}); ok {
								if gained, ok := kills["gained"].(float64); ok && gained > 0 {
									printed = true
								}
							}
						}
					}
					if !printed {
						fmt.Printf("\n  %s: 0 gains this %s\n", formatBossName(metricNorm), period)
					}
				} else {
					fmt.Printf("\n  Unknown metric '%s'. Use skill names (ranged, slayer) or boss names (vorkath, zulrah).\n", metric)
				}
			} else if gainsData["skills"] == nil && gainsData["bosses"] == nil {
				fmt.Println("  No gains this period.")
			}
		}
		return nil
	},
}

var playerUpdateCmd = &cobra.Command{
	Use:     "update [username]",
	Short:   "Trigger a stats refresh for a player",
	Args:    cobra.ExactArgs(1),
	Example: `  wom player update Zezima`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		data, err := client.UpdatePlayer(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("Updated %s\n", args[0])
		}
		return nil
	},
}

var playerSearchCmd = &cobra.Command{
	Use:     "search [query]",
	Short:   "Search for players by partial username",
	Args:    cobra.ExactArgs(1),
	Example: `  wom player search zezi`,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		client := api.NewClient()
		results, err := client.SearchPlayers(args[0], limit)
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
			fmt.Printf("No players found matching '%s'\n", args[0])
			os.Exit(3)
		}
		for _, r := range results {
			if p, ok := r.(map[string]interface{}); ok {
				name, _ := p["displayName"].(string)
				ptype, _ := p["type"].(string)
				exp, _ := p["exp"].(float64)
				fmt.Printf("  %s (%s) — %s XP\n", name, ptype, formatNumber(exp))
			}
		}
		return nil
	},
}

var playerAchievementsCmd = &cobra.Command{
	Use:     "achievements [username]",
	Short:   "Show achievements for a player",
	Args:    cobra.ExactArgs(1),
	Example: `  wom player achievements Zezima`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient()
		results, err := client.GetPlayerAchievements(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(3)
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		fmt.Printf("Achievements for %s:\n", args[0])
		for _, r := range results {
			if a, ok := r.(map[string]interface{}); ok {
				name, _ := a["name"].(string)
				fmt.Printf("  ✅ %s\n", name)
			}
		}
		return nil
	},
}

func init() {
	playerGainsCmd.Flags().String("period", "week", "Time period: day, week, month, year")
	playerGainsCmd.Flags().String("metric", "", "Filter to specific skill or boss (e.g., vorkath, ranged)")
	playerSearchCmd.Flags().Int("limit", 10, "Max results")

	playerCmd.AddCommand(playerLookupCmd)
	playerCmd.AddCommand(playerGainsCmd)
	playerCmd.AddCommand(playerUpdateCmd)
	playerCmd.AddCommand(playerSearchCmd)
	playerCmd.AddCommand(playerAchievementsCmd)
}

func formatNumber(n float64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", n/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.0fK", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}

// formatEHB formats EHB/EHP values with 1 decimal
func formatEHB(n float64) string {
	return fmt.Sprintf("%.1f", n)
}

// formatBossName converts API snake_case to readable names.
// "chambers_of_xeric_challenge_mode" → "Chambers of Xeric (CM)"
func formatBossName(name string) string {
	replacements := map[string]string{
		"chambers_of_xeric":                "Chambers of Xeric",
		"chambers_of_xeric_challenge_mode": "Chambers of Xeric (CM)",
		"theatre_of_blood":                 "Theatre of Blood",
		"theatre_of_blood_hard_mode":       "Theatre of Blood (HM)",
		"tombs_of_amascut":                 "Tombs of Amascut",
		"tombs_of_amascut_expert":          "Tombs of Amascut (Expert)",
		"corporeal_beast":                  "Corporeal Beast",
		"dagannoth_prime":                  "Dagannoth Prime",
		"dagannoth_rex":                    "Dagannoth Rex",
		"dagannoth_supreme":                "Dagannoth Supreme",
		"alchemical_hydra":                 "Alchemical Hydra",
		"thermonuclear_smoke_devil":        "Thermonuclear Smoke Devil",
		"kalphite_queen":                   "Kalphite Queen",
		"giant_mole":                       "Giant Mole",
		"king_black_dragon":                "King Black Dragon",
		"phantom_muspah":                   "Phantom Muspah",
		"duke_sucellus":                    "Duke Sucellus",
		"the_leviathan":                    "The Leviathan",
		"the_whisperer":                    "The Whisperer",
		"vardorvis":                        "Vardorvis",
		"general_graardor":                 "General Graardor",
		"commander_zilyana":                "Commander Zilyana",
		"kril_tsutsaroth":                  "K'ril Tsutsaroth",
	}
	if pretty, ok := replacements[name]; ok {
		return pretty
	}
	// Fallback: capitalize each word, replace underscores
	words := strings.Split(name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
