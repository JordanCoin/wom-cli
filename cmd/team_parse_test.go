package cmd

import (
	"strings"
	"testing"
)

func TestATeamIsNameThenPlayers(t *testing.T) {
	teams, err := parseTeams([]string{"Team Alpha=Doe Matic,Uka36"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(teams) != 1 || teams[0].Name != "Team Alpha" {
		t.Fatalf("name not parsed: %#v", teams)
	}
	// A space inside a player name must survive: real OSRS names have them.
	if len(teams[0].Participants) != 2 || teams[0].Participants[0] != "Doe Matic" {
		t.Fatalf("players not parsed: %#v", teams[0].Participants)
	}
}

func TestSurroundingSpaceIsTrimmedFromNamesAndPlayers(t *testing.T) {
	teams, err := parseTeams([]string{"  Team Bravo = Zezima ,  Lynx Titan  "})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if teams[0].Name != "Team Bravo" {
		t.Fatalf("team name not trimmed: %q", teams[0].Name)
	}
	if teams[0].Participants[1] != "Lynx Titan" {
		t.Fatalf("player not trimmed: %q", teams[0].Participants[1])
	}
}

func TestATeamMissingItsPlayersSaysHowToWriteIt(t *testing.T) {
	_, err := parseTeams([]string{"Team Alpha"})
	if err == nil {
		t.Fatal("a team with no '=' must be refused")
	}
	if !strings.Contains(err.Error(), "Name=player1") {
		t.Fatalf("the error should show the shape, got: %s", err)
	}
}

func TestTheSameTeamNameTwiceIsRefused(t *testing.T) {
	// WOM keys a team by name, so a duplicate silently merges two rosters
	// into one team rather than erroring.
	_, err := parseTeams([]string{"Alpha=Zezima", "alpha=Uka36"})
	if err == nil {
		t.Fatal("a repeated team name must be refused, case-insensitively")
	}
}

func TestATeamWithAnEmptyRosterIsRefused(t *testing.T) {
	if _, err := parseTeams([]string{"Alpha=  ,  "}); err == nil {
		t.Fatal("a team listing no players must be refused")
	}
}

func TestNoTeamFlagsMeansNoTeams(t *testing.T) {
	teams, err := parseTeams(nil)
	if err != nil || teams != nil {
		t.Fatalf("absent --team must stay absent, got %#v / %v", teams, err)
	}
}
