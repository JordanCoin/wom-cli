package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client wraps the WiseOldMan API v2.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a WOM API client.
func NewClient() *Client {
	return &Client{
		BaseURL: "https://api.wiseoldman.net/v2",
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ── Players ──────────────────────────────────────────────────────────

// GetPlayer fetches player stats by username.
func (c *Client) GetPlayer(username string) (map[string]interface{}, error) {
	return c.getMap("/players/" + url.PathEscape(username))
}

// UpdatePlayer triggers a stats refresh for a player.
func (c *Client) UpdatePlayer(username string) (map[string]interface{}, error) {
	return c.postMap("/players/"+url.PathEscape(username), nil)
}

// GetPlayerGains fetches XP gains for a player over a period.
// Periods: day, week, month, year
func (c *Client) GetPlayerGains(username, period string) (map[string]interface{}, error) {
	return c.getMap("/players/" + url.PathEscape(username) + "/gained?period=" + period)
}

// GetPlayerAchievements fetches achievements for a player.
func (c *Client) GetPlayerAchievements(username string) ([]interface{}, error) {
	return c.getArray("/players/" + url.PathEscape(username) + "/achievements")
}

// GetPlayerRecords fetches personal records for a player.
func (c *Client) GetPlayerRecords(username, period, metric string) ([]interface{}, error) {
	path := "/players/" + url.PathEscape(username) + "/records"
	params := url.Values{}
	if period != "" {
		params.Set("period", period)
	}
	if metric != "" {
		params.Set("metric", metric)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return c.getArray(path)
}

// SearchPlayers searches for players by partial username.
func (c *Client) SearchPlayers(query string, limit int) ([]interface{}, error) {
	path := fmt.Sprintf("/players/search?username=%s&limit=%d", url.QueryEscape(query), limit)
	return c.getArray(path)
}

// ── Groups ───────────────────────────────────────────────────────────

// GetGroup fetches group info by ID.
func (c *Client) GetGroup(groupID string) (map[string]interface{}, error) {
	return c.getMap("/groups/" + groupID)
}

// GetGroupGains fetches the gains leaderboard for a group.
func (c *Client) GetGroupGains(groupID, metric, period string, limit int) ([]interface{}, error) {
	path := fmt.Sprintf("/groups/%s/gained?metric=%s&period=%s&limit=%d",
		groupID, url.QueryEscape(metric), url.QueryEscape(period), limit)
	return c.getArray(path)
}

// GetGroupHiscores fetches hiscores for a group.
func (c *Client) GetGroupHiscores(groupID, metric string, limit int) ([]interface{}, error) {
	path := fmt.Sprintf("/groups/%s/hiscores?metric=%s&limit=%d",
		groupID, url.QueryEscape(metric), limit)
	return c.getArray(path)
}

// GetGroupCompetitions fetches competitions for a group.
func (c *Client) GetGroupCompetitions(groupID string) ([]interface{}, error) {
	return c.getArray("/groups/" + groupID + "/competitions")
}

// GetGroupMembers fetches all members of a group.
func (c *Client) GetGroupMembers(groupID string) ([]interface{}, error) {
	return c.getArray("/groups/" + groupID + "/members")
}

// ── Competitions ─────────────────────────────────────────────────────

// CreateCompetition creates a new competition.
// TeamSpec is one side of a team competition: a name and the players on it.
type TeamSpec struct {
	Name         string
	Participants []string
}

// buildCompetitionBody is the POST /competitions payload, built without any I/O
// so it can be tested on its own.
//
// WOM treats `participants` and `teams` as alternatives, not as a pair: sending
// both is rejected, and sending neither creates a competition nobody is in. The
// choice between them is what makes a competition classic or team, so it is
// checked here rather than left to the API to refuse after the fact.
func buildCompetitionBody(title, metric, startsAt, endsAt string, participants []string, teams []TeamSpec, groupID, groupVerificationCode string) (map[string]interface{}, error) {
	if len(participants) > 0 && len(teams) > 0 {
		return nil, fmt.Errorf("a competition is either classic or team: pass --participants or --team, not both")
	}

	body := map[string]interface{}{
		"title":    title,
		"metric":   metric,
		"startsAt": startsAt,
		"endsAt":   endsAt,
	}

	if len(participants) > 0 {
		body["participants"] = participants
	}

	if len(teams) > 0 {
		encoded := make([]map[string]interface{}, 0, len(teams))
		for _, t := range teams {
			if t.Name == "" {
				return nil, fmt.Errorf("every team needs a name")
			}
			if len(t.Participants) == 0 {
				return nil, fmt.Errorf("team %q has no players", t.Name)
			}
			encoded = append(encoded, map[string]interface{}{
				"name":         t.Name,
				"participants": t.Participants,
			})
		}
		body["teams"] = encoded
	}

	if groupID != "" {
		groupIDNum, err := strconv.ParseInt(groupID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid group ID %q: %s", groupID, err)
		}
		body["groupId"] = groupIDNum
		body["groupVerificationCode"] = groupVerificationCode
	}

	return body, nil
}

// CreateCompetition creates a classic competition (participants) or a team
// competition (teams). A group id plus verification code attaches it to the
// group, which is what puts it on the group's page; without them it exists but
// is only reachable by its own id.
func (c *Client) CreateCompetition(title, metric, startsAt, endsAt string, participants []string, teams []TeamSpec, groupID string, groupVerificationCode string) (map[string]interface{}, error) {
	body, err := buildCompetitionBody(title, metric, startsAt, endsAt, participants, teams, groupID, groupVerificationCode)
	if err != nil {
		return nil, err
	}
	return c.postMap("/competitions", body)
}

// GetCompetition fetches competition details.
func (c *Client) GetCompetition(competitionID string) (map[string]interface{}, error) {
	return c.getMap("/competitions/" + competitionID)
}

// GetCompetitionStandings fetches the current standings.
func (c *Client) GetCompetitionStandings(competitionID string) ([]interface{}, error) {
	data, err := c.getMap("/competitions/" + competitionID)
	if err != nil {
		return nil, err
	}
	if participations, ok := data["participations"].([]interface{}); ok {
		return participations, nil
	}
	return nil, fmt.Errorf("no participations data found")
}

// EditCompetition updates a competition's title, end date, or participants.
func (c *Client) EditCompetition(competitionID string, updates map[string]interface{}) (map[string]interface{}, error) {
	return c.putMap("/competitions/"+competitionID, updates)
}

// AddParticipants adds players to a competition.
func (c *Client) AddParticipants(competitionID string, participants []string, verificationCode string) error {
	body := map[string]interface{}{
		"verificationCode": verificationCode,
		"participants":     participants,
	}
	_, err := c.postMap("/competitions/"+competitionID+"/participants", body)
	return err
}

// RemoveParticipants removes players from a competition.
func (c *Client) RemoveParticipants(competitionID string, participants []string, verificationCode string) error {
	body := map[string]interface{}{
		"verificationCode": verificationCode,
		"participants":     participants,
	}
	_, err := c.deleteMap("/competitions/"+competitionID+"/participants", body)
	return err
}

// UpdateAllParticipants queues a stat refresh for all outdated participants.
func (c *Client) UpdateAllParticipants(competitionID, verificationCode string) error {
	body := map[string]interface{}{
		"verificationCode": verificationCode,
	}
	_, err := c.postMap("/competitions/"+competitionID+"/update-all", body)
	return err
}

// DeleteCompetition deletes a competition.
func (c *Client) DeleteCompetition(competitionID, verificationCode string) error {
	body := map[string]interface{}{
		"verificationCode": verificationCode,
	}
	_, err := c.deleteMap("/competitions/"+competitionID, body)
	return err
}

// ── HTTP helpers ─────────────────────────────────────────────────────

func (c *Client) getMap(path string) (map[string]interface{}, error) {
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return result, nil
}

func (c *Client) getArray(path string) ([]interface{}, error) {
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return result, nil
}

func (c *Client) postMap(path string, body interface{}) (map[string]interface{}, error) {
	data, err := c.doRequest("POST", path, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return result, nil
}

func (c *Client) putMap(path string, body interface{}) (map[string]interface{}, error) {
	data, err := c.doRequest("PUT", path, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return result, nil
}

func (c *Client) deleteMap(path string, body interface{}) (map[string]interface{}, error) {
	data, err := c.doRequest("DELETE", path, body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return result, nil
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wom-cli (github.com/JordanCoin/wom-cli)")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found — player/group may not be tracked. Visit wiseoldman.net to start tracking")
	}
	if resp.StatusCode == 400 {
		var errResp map[string]interface{}
		if json.Unmarshal(data, &errResp) == nil {
			if msg, ok := errResp["message"].(string); ok {
				return nil, fmt.Errorf("%s", msg)
			}
		}
		return nil, fmt.Errorf("bad request: %s", string(data))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
