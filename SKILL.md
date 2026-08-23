---
name: wiseoldman
description: "User wants to look up OSRS player stats, XP gains, group leaderboards, hiscores, or manage WOM competitions"
version: 0.3.0
---

# Skill: WiseOldMan CLI

**Use the `wom` CLI via `exec_skill` for OSRS player lookups, gains tracking, group leaderboards, hiscores, and competitions.**

Our group ID is **5165**.

**Team competitions:** `competition create` takes either `--participants` (classic)
or repeated `--team "Name=player1,player2"` (team), never both. A clan bingo uses
the team form: every participation then carries `teamName`, so `competition view`
gives per-team standings with no separate roster mapping.

## Player Lookups

```json
exec_skill({"skill": "wom", "args": "player lookup 'Doe Matic'"})
exec_skill({"skill": "wom", "args": "player gains 'Doe Matic' --period week"})
exec_skill({"skill": "wom", "args": "player gains 'Doe Matic' --period month --metric slayer"})
exec_skill({"skill": "wom", "args": "player search 'doe'"})
exec_skill({"skill": "wom", "args": "player achievements 'Doe Matic'"})
exec_skill({"skill": "wom", "args": "player update 'Doe Matic'"})
```

- `lookup` — combat level, total XP, total level, top bosses, EHP/EHB
- `gains` — XP and KC gains over a period (day, week, month, year)
- `--metric` filters gains to a specific skill or boss
- `update` triggers a fresh stat pull from OSRS hiscores

## Group Leaderboards (Gains)

Rankings by XP/KC **gained** over a time period:

```json
exec_skill({"skill": "wom", "args": "group leaderboard 5165 --metric overall --period week --top 10"})
exec_skill({"skill": "wom", "args": "group leaderboard 5165 --metric runecrafting --period month --top 5"})
exec_skill({"skill": "wom", "args": "group leaderboard 5165 --metric vorkath --period week --top 5"})
```

- Metrics: any skill (overall, attack, runecrafting, etc.) or boss (vorkath, zulrah, etc.)
- Periods: day, week, month, year

## Group Hiscores (Total Stats)

Rankings by **total stats** (all-time XP, level, or KC):

```json
exec_skill({"skill": "wom", "args": "group hiscores 5165 --metric overall --top 10"})
exec_skill({"skill": "wom", "args": "group hiscores 5165 --metric overall --sort-by level --top 20"})
exec_skill({"skill": "wom", "args": "group hiscores 5165 --metric slayer --top 10"})
```

- Default sort is by XP. Use `--sort-by level` to rank by total level instead.
- `--sort-by level` is useful for finding players closest to maxing (max total level is 2376 with Sailing).
- For "closest to maxing" queries, use `--sort-by level --top 30` and look for players below 2376.

## Group Info & Members

```json
exec_skill({"skill": "wom", "args": "group info 5165"})
exec_skill({"skill": "wom", "args": "group members 5165"})
exec_skill({"skill": "wom", "args": "group competitions 5165"})
```

## Important Rules

- Always quote player names: `'Doe Matic'` not `Doe Matic`
- Use `--json` when you need to parse the output programmatically
- Group ID 5165 is our clan (Me so scape)
- Max total level is 2376 (24 skills including Sailing)
