package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAClassicCompetitionSendsParticipantsAndNoTeams(t *testing.T) {
	body, err := buildCompetitionBody("RC SOTW", "runecrafting", "s", "e",
		[]string{"Zezima", "Lynx Titan"}, nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, ok := body["teams"]; ok {
		t.Fatal("a classic competition must not carry a teams key")
	}
	got, _ := body["participants"].([]string)
	if len(got) != 2 || got[0] != "Zezima" {
		t.Fatalf("participants not passed through: %#v", body["participants"])
	}
}

func TestATeamCompetitionSendsTeamsAndNoParticipants(t *testing.T) {
	body, err := buildCompetitionBody("Summer Bingo", "ehb", "s", "e", nil,
		[]TeamSpec{
			{Name: "Team Alpha", Participants: []string{"Doe Matic", "Uka36"}},
			{Name: "Team Bravo", Participants: []string{"Zezima"}},
		}, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, ok := body["participants"]; ok {
		t.Fatal("a team competition must not carry a participants key")
	}

	// Assert on the wire form, because that is what WOM actually reads.
	raw, err := json.Marshal(body["teams"])
	if err != nil {
		t.Fatalf("teams did not marshal: %s", err)
	}
	want := `[{"name":"Team Alpha","participants":["Doe Matic","Uka36"]},{"name":"Team Bravo","participants":["Zezima"]}]`
	if string(raw) != want {
		t.Fatalf("wrong wire shape\n got: %s\nwant: %s", raw, want)
	}
}

func TestParticipantsAndTeamsTogetherIsRefusedBeforeTheRequest(t *testing.T) {
	_, err := buildCompetitionBody("x", "ehb", "s", "e",
		[]string{"Zezima"}, []TeamSpec{{Name: "A", Participants: []string{"B"}}}, "", "")
	if err == nil {
		t.Fatal("sending both must be refused here, not left for WOM to reject")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Fatalf("the error should say they are alternatives, got: %s", err)
	}
}

func TestATeamWithNoPlayersIsRefused(t *testing.T) {
	if _, err := buildCompetitionBody("x", "ehb", "s", "e", nil,
		[]TeamSpec{{Name: "Empty"}}, "", ""); err == nil {
		t.Fatal("a team with no players would create a side nobody can score for")
	}
}

func TestAGroupIdIsSentAsANumberAlongsideTeams(t *testing.T) {
	body, err := buildCompetitionBody("x", "ehb", "s", "e", nil,
		[]TeamSpec{{Name: "A", Participants: []string{"Zezima"}}}, "5165", "code-here")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// Regression: d193156 fixed groupId being sent as a string. A team
	// competition takes the same path and must not reintroduce it.
	if _, ok := body["groupId"].(int64); !ok {
		t.Fatalf("groupId must be numeric, got %T", body["groupId"])
	}
	if body["groupVerificationCode"] != "code-here" {
		t.Fatal("the verification code must travel with the group id")
	}
}

func TestABadGroupIdIsReported(t *testing.T) {
	if _, err := buildCompetitionBody("x", "ehb", "s", "e",
		[]string{"Zezima"}, nil, "not-a-number", ""); err == nil {
		t.Fatal("a non-numeric group id must be refused")
	}
}
