package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

type teamIDRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type standingRow struct {
	Position int64     `json:"position"`
	Team     teamIDRef `json:"team"`
	Played   int64     `json:"played"`
	Won      int64     `json:"won"`
	Drawn    int64     `json:"drawn"`
	Lost     int64     `json:"lost"`
	GF       int64     `json:"gf"`
	GA       int64     `json:"ga"`
	GD       int64     `json:"gd"`
	Points   int64     `json:"points"`
}

type standingsResponse struct {
	League    string        `json:"league"`
	Season    string        `json:"season"`
	Standings []standingRow `json:"standings"`
}

type fixtureRow struct {
	ID        int64     `json:"id"`
	External  *string   `json:"externalId"`
	Kickoff   string    `json:"kickoff"`
	Status    string    `json:"status"`
	Home      teamIDRef `json:"home"`
	Away      teamIDRef `json:"away"`
	HomeGoals *int64    `json:"homeGoals"`
	AwayGoals *int64    `json:"awayGoals"`
}

type fixturesResponse struct {
	League   string       `json:"league"`
	Season   string       `json:"season"`
	Fixtures []fixtureRow `json:"fixtures"`
}

var errUnknownLeague = errors.New("unknown league")

func handleLeagueStandings(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		comp, err := lookupCompetition(r.Context(), st, r.PathValue("leagueId"))
		if err != nil {
			writeLeagueError(w, err)
			return
		}
		resp, err := buildStandings(r.Context(), st, comp)
		if err != nil {
			writeLeagueError(w, err)
			return
		}
		writeJSON(w, resp)
	}
}

func buildStandings(ctx context.Context, st *store.Store, comp db.Competition) (standingsResponse, error) {
	season, err := st.GetLatestSeasonByCompetition(ctx, comp.ID)
	if errors.Is(err, sql.ErrNoRows) {
		season = ""
	} else if err != nil {
		return standingsResponse{}, err
	}
	rows, err := st.GetStandingsByCompetition(ctx, comp.ID)
	if err != nil {
		return standingsResponse{}, err
	}
	out := standingsResponse{League: comp.Code, Season: season, Standings: make([]standingRow, 0, len(rows))}
	for i, row := range rows {
		out.Standings = append(out.Standings, standingRow{
			Position: int64(i + 1),
			Team:     teamIDRef{ID: row.TeamID, Name: row.TeamName},
			Played:   row.Played,
			Won:      int64(row.Won),
			Drawn:    int64(row.Drawn),
			Lost:     int64(row.Lost),
			GF:       int64(row.GoalsFor),
			GA:       int64(row.GoalsAgainst),
			GD:       row.GoalDiff,
			Points:   row.Points,
		})
	}
	return out, nil
}

func handleLeagueFixtures(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		comp, err := lookupCompetition(r.Context(), st, r.PathValue("leagueId"))
		if err != nil {
			writeLeagueError(w, err)
			return
		}
		season := r.URL.Query().Get("season")
		if season == "" {
			season, err = st.GetLatestSeasonByCompetition(r.Context(), comp.ID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				writeLeagueError(w, err)
				return
			}
		}
		resp, err := buildFixtures(r.Context(), st, comp, season)
		if err != nil {
			writeLeagueError(w, err)
			return
		}
		writeJSON(w, resp)
	}
}

func buildFixtures(ctx context.Context, st *store.Store, comp db.Competition, season string) (fixturesResponse, error) {
	rows, err := st.ListFixturesByLeagueAndSeason(ctx, db.ListFixturesByLeagueAndSeasonParams{
		CompetitionID: comp.ID,
		Season:        season,
	})
	if err != nil {
		return fixturesResponse{}, err
	}
	out := fixturesResponse{League: comp.Code, Season: season, Fixtures: make([]fixtureRow, 0, len(rows))}
	for _, row := range rows {
		out.Fixtures = append(out.Fixtures, fixtureRow{
			ID:        row.ID,
			External:  nullStringPtr(row.ExternalID),
			Kickoff:   row.UtcKickoff,
			Status:    row.Status,
			Home:      teamIDRef{ID: row.HomeTeamID, Name: row.HomeTeamName},
			Away:      teamIDRef{ID: row.AwayTeamID, Name: row.AwayTeamName},
			HomeGoals: nullInt64Ptr(row.HomeGoals),
			AwayGoals: nullInt64Ptr(row.AwayGoals),
		})
	}
	return out, nil
}

func lookupCompetition(ctx context.Context, st *store.Store, code string) (db.Competition, error) {
	comp, err := st.GetCompetitionByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return db.Competition{}, errUnknownLeague
	}
	return comp, err
}

func writeLeagueError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnknownLeague) {
		http.Error(w, `{"error":"unknown league"}`, http.StatusNotFound)
		return
	}
	http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
}
