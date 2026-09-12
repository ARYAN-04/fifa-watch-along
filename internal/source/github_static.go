package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	githubStaticBaseURL   = "https://raw.githubusercontent.com/openfootball/football.json/master"
	defaultGitHubSeason   = "2025-26"
	githubStaticUserAgent = "fifa-hub"
)

var githubStaticLeagueFiles = map[string]string{
	"PL": "en.1",
}

type GitHubStatic struct {
	BaseURL    string
	Season     string
	HTTPClient *http.Client
}

func NewGitHubStatic() *GitHubStatic {
	return &GitHubStatic{
		BaseURL:    githubStaticBaseURL,
		Season:     defaultGitHubSeason,
		HTTPClient: http.DefaultClient,
	}
}

type ofRoot struct {
	Matches []ofMatch `json:"matches"`
}

type ofMatch struct {
	Date  string          `json:"date"`
	Time  string          `json:"time"`
	Team1 string          `json:"team1"`
	Team2 string          `json:"team2"`
	Score json.RawMessage `json:"score"`
}

func (g *GitHubStatic) LiveMatches(ctx context.Context, leagueCode string) ([]Match, error) {
	file, ok := githubStaticLeagueFiles[leagueCode]
	if !ok {
		return nil, fmt.Errorf("github static: unsupported league %q", leagueCode)
	}
	endpoint := fmt.Sprintf("%s/%s/%s.json", g.base(), g.season(), file)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("github static: build request: %w", err)
	}
	req.Header.Set("User-Agent", githubStaticUserAgent)
	client := g.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github static %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github static %s: status %d: %s", endpoint, resp.StatusCode, body)
	}
	var root ofRoot
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("github static %s: decode response: %w", endpoint, err)
	}
	matches := make([]Match, 0, len(root.Matches))
	for _, m := range root.Matches {
		matches = append(matches, toSourceMatch(g.season(), m))
	}
	return matches, nil
}

func toSourceMatch(season string, m ofMatch) Match {
	homeGoals, awayGoals, played := parseOFScore(m.Score)
	status := "SCHEDULED"
	if played {
		status = "FINISHED"
	}
	kickoff := m.Date + "T00:00:00Z"
	if m.Time != "" {
		kickoff = fmt.Sprintf("%sT%s:00Z", m.Date, m.Time)
	}
	return Match{
		ExternalID: season + "/" + m.Date + "/" + m.Team1 + "/" + m.Team2,
		HomeTeam:   m.Team1,
		AwayTeam:   m.Team2,
		HomeGoals:  homeGoals,
		AwayGoals:  awayGoals,
		Status:     status,
		UTCDate:    kickoff,
	}
}

func parseOFScore(raw json.RawMessage) (home, away *int, played bool) {
	var pair []int
	if err := json.Unmarshal(raw, &pair); err == nil && len(pair) == 2 {
		return &pair[0], &pair[1], true
	}
	var shaped struct {
		FT []int `json:"ft"`
	}
	if err := json.Unmarshal(raw, &shaped); err == nil && len(shaped.FT) == 2 {
		return &shaped.FT[0], &shaped.FT[1], true
	}
	return nil, nil, false
}

func (g *GitHubStatic) base() string {
	if g.BaseURL == "" {
		return githubStaticBaseURL
	}
	return g.BaseURL
}

func (g *GitHubStatic) season() string {
	if g.Season == "" {
		return defaultGitHubSeason
	}
	return g.Season
}
