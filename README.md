# wom — WiseOldMan CLI

CLI tool for the [WiseOldMan](https://wiseoldman.net) OSRS API. Look up player stats, track gains, manage group leaderboards, and run competitions.

Built for [AFK Mod](https://afkmod.app) — agent-friendly with `--json` output and meaningful exit codes.

## Install

```bash
curl -sL https://github.com/JordanCoin/wom-cli/releases/latest/download/wom-linux-amd64 -o /usr/local/bin/wom
chmod +x /usr/local/bin/wom
```

## Player Commands

```bash
# Look up a player (stats, combat, EHP/EHB, top boss KCs)
wom player lookup "Doe Matic"

# XP + boss KC gains this week
wom player gains "Doe Matic" --period week

# Trigger a stat refresh from OSRS hiscores
wom player update "Doe Matic"

# Search by partial name
wom player search "doe" --limit 5

# View achievements
wom player achievements "Doe Matic"
```

## Group Commands

```bash
# Group info
wom group info 5165

# XP leaderboard — any metric, any period
wom group leaderboard 5165 --metric overall --period week --top 10
wom group leaderboard 5165 --metric runecrafting --period month --top 5

# Boss efficiency leaderboard
wom group leaderboard 5165 --metric ehb --period week --top 10

# Boss-specific leaderboard
wom group leaderboard 5165 --metric vorkath --period week --top 5

# Group hiscores (total stats, not gains)
wom group hiscores 5165 --metric overall --top 10

# List all members
wom group members 5165

# List competitions
wom group competitions 5165
```

## Competition Commands

```bash
# Create a Skill of the Week
wom competition create \
  --title "RC SOTW" \
  --metric runecrafting \
  --starts "2026-03-20T00:00:00Z" \
  --ends "2026-03-27T00:00:00Z" \
  --participants "Doe Matic,Uka36,pollieolly"

# Create a group competition (all members auto-included)
wom competition create \
  --title "PVM BOTW" \
  --metric vorkath \
  --starts "2026-03-20T00:00:00Z" \
  --ends "2026-03-27T00:00:00Z" \
  --group-id 5165 \
  --verification-code "123-456-789"

# Create a TEAM competition (the shape a clan bingo uses)
# Repeat --team for each side. Every participation then carries teamName,
# so standings can be summed per team without a separate roster mapping.
wom competition create \
  --title "Summer Bingo" \
  --metric ehb \
  --starts "2026-06-13T00:00:00Z" \
  --ends "2026-06-27T00:00:00Z" \
  --group-id 5165 \
  --verification-code "123-456-789" \
  --team "Team Alpha=Doe Matic,Uka36" \
  --team "Team Bravo=pollieolly,Zezima"

# --participants and --team are alternatives, never both: passing both is
# refused before the request, because WOM rejects it too.

# View standings
wom competition view 12345

# Add/remove participants mid-competition
wom competition add-participants 12345 --players "NewPlayer1,NewPlayer2" --verification-code "123-456-789"
wom competition remove-participants 12345 --players "BadPlayer" --verification-code "123-456-789"

# Edit title or end date
wom competition edit 12345 --title "RC SOTW Extended" --ends "2026-03-30T00:00:00Z" --verification-code "123-456-789"

# Refresh all participant stats
wom competition update-all 12345 --verification-code "123-456-789"

# Delete
wom competition delete 12345 --verification-code "123-456-789"
```

### Keeping the verification code out of argv

A group's verification code authorises editing and deleting every competition
the group owns, and it does not rotate on its own. Every command above takes it
from `$WOM_VERIFICATION_CODE` when `--verification-code` is absent, so a script
or an agent never has to put it on a command line, where it would land in `ps`,
in shell history, and in whatever log the caller keeps of the commands it ran.
`--group-id` falls back to `$WOM_GROUP_ID` the same way; that one is not a
secret, it is only tedious to repeat.

An explicit flag always beats the environment, so a single invocation can still
override an ambient default.

```bash
export WOM_GROUP_ID=5165
export WOM_VERIFICATION_CODE="123-456-789"

wom competition create --title "Summer Bingo" --metric ehb \
  --starts "2026-09-25T00:00:00Z" --ends "2026-10-04T00:00:00Z" \
  --team "Team Alpha=Doe Matic,Uka36" --json
```

A supplied code is never echoed back, on either the human or the `--json` path:
you already have it, and printing it would only copy it somewhere new. The one
code that *is* printed is the one a standalone competition mints for itself,
because that output is the only place it ever appears.

## JSON Output

Add `--json` to any command for structured output:

```bash
wom player lookup "Doe Matic" --json
wom group leaderboard 5165 --metric ehb --period week --top 5 --json
```

## Metrics

**Skills:** overall, attack, defence, strength, hitpoints, ranged, prayer, magic, cooking, woodcutting, fletching, fishing, firemaking, crafting, smithing, mining, herblore, agility, thieving, slayer, farming, runecrafting, hunter, construction, sailing

**Bosses:** zulrah, vorkath, corporeal_beast, nex, cerberus, kraken, alchemical_hydra, theatre_of_blood, chambers_of_xeric, tombs_of_amascut, the_leviathan, duke_sucellus, vardorvis, the_whisperer, phantom_muspah, and [many more](https://docs.wiseoldman.net/metrics)

**Computed:** ehp (Efficient Hours Played), ehb (Efficient Hours Bossed)

**Periods:** day, week, month, year

## License

MIT
