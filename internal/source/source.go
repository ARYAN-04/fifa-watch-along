package source

import "context"

type Match struct {
	ExternalID string
	HomeTeam   string
	AwayTeam   string
	HomeGoals  *int
	AwayGoals  *int
	Minute     *int
	Status     string
	UTCDate    string
}

type DataSource interface {
	LiveMatches(ctx context.Context, leagueCode string) ([]Match, error)
}

// NormalizeStatus maps football-data.org match statuses to the three
// canonical statuses stored in football.db: SCHEDULED, LIVE, FINISHED.
func NormalizeStatus(s string) string {
	switch s {
	case "IN_PLAY", "PAUSED":
		return "LIVE"
	case "FINISHED", "AET", "PEN":
		return "FINISHED"
	default:
		return "SCHEDULED"
	}
}
