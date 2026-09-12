package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

var errUnknownTeam = errors.New("unknown team")

type formEntry struct {
	Result   string `json:"result"`
	Opponent string `json:"opponent"`
	Date     string `json:"date"`
	Score    string `json:"score"`
}

type compareTeam struct {
	ID   int64       `json:"id"`
	Name string      `json:"name"`
	Form []formEntry `json:"form"`
	Elo  *float64    `json:"elo"`
}

type teamIDName struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type h2hMatch struct {
	Date      string     `json:"date"`
	Season    string     `json:"season"`
	Home      teamIDName `json:"home"`
	Away      teamIDName `json:"away"`
	HomeGoals *int64     `json:"homeGoals"`
	AwayGoals *int64     `json:"awayGoals"`
}

type h2hSummary struct {
	Played   int        `json:"played"`
	HomeWins int        `json:"homeWins"`
	AwayWins int        `json:"awayWins"`
	Draws    int        `json:"draws"`
	AvgGoals float64    `json:"avgGoals"`
	Matches  []h2hMatch `json:"matches"`
}

type compareResponse struct {
	Home compareTeam `json:"home"`
	Away compareTeam `json:"away"`
	H2H  h2hSummary  `json:"h2h"`
}

func handleTeamCompare(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		homeID, err := strconv.ParseInt(r.URL.Query().Get("home"), 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid home id"}`, http.StatusBadRequest)
			return
		}
		awayID, err := strconv.ParseInt(r.URL.Query().Get("away"), 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid away id"}`, http.StatusBadRequest)
			return
		}
		resp, err := buildCompare(r.Context(), st, homeID, awayID)
		if errors.Is(err, errUnknownTeam) {
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("api: teams/compare: %v", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	}
}

func buildCompare(ctx context.Context, st *store.Store, homeID, awayID int64) (compareResponse, error) {
	var resp compareResponse
	var err error
	if resp.Home, err = buildCompareTeam(ctx, st, homeID); err != nil {
		return resp, err
	}
	if resp.Away, err = buildCompareTeam(ctx, st, awayID); err != nil {
		return resp, err
	}
	rows, err := st.ListFinishedBetweenTeams(ctx, db.ListFinishedBetweenTeamsParams{TeamA: homeID, TeamB: awayID})
	if err != nil {
		return resp, fmt.Errorf("list h2h matches: %w", err)
	}
	resp.H2H = buildH2H(rows, homeID, awayID)
	return resp, nil
}

func buildCompareTeam(ctx context.Context, st *store.Store, teamID int64) (compareTeam, error) {
	team, err := st.GetTeamByID(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		return compareTeam{}, errUnknownTeam
	}
	if err != nil {
		return compareTeam{}, fmt.Errorf("get team %d: %w", teamID, err)
	}
	form, err := buildForm(ctx, st, teamID)
	if err != nil {
		return compareTeam{}, err
	}
	elo, err := fetchElo(ctx, st, teamID)
	if err != nil {
		return compareTeam{}, err
	}
	return compareTeam{ID: team.ID, Name: team.Name, Form: form, Elo: elo}, nil
}

func buildForm(ctx context.Context, st *store.Store, teamID int64) ([]formEntry, error) {
	rows, err := st.ListLastFinishedByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list form for team %d: %w", teamID, err)
	}
	form := make([]formEntry, 0, len(rows))
	for _, row := range rows {
		hg, ag := derefInt(row.HomeGoals), derefInt(row.AwayGoals)
		opponent := row.AwayName
		gf, ga := hg, ag
		if row.AwayTeamID == teamID {
			opponent = row.HomeName
			gf, ga = ag, hg
		}
		form = append(form, formEntry{
			Result:   resultLetter(gf, ga),
			Opponent: opponent,
			Date:     row.UtcKickoff,
			Score:    fmt.Sprintf("%d-%d", hg, ag),
		})
	}
	return form, nil
}

func buildH2H(rows []db.ListFinishedBetweenTeamsRow, homeID, awayID int64) h2hSummary {
	summary := h2hSummary{Played: len(rows), Matches: make([]h2hMatch, 0, min(len(rows), 10))}
	var totalGoals int64
	for i, row := range rows {
		hg, ag := derefInt(row.HomeGoals), derefInt(row.AwayGoals)
		totalGoals += hg + ag
		switch {
		case hg > ag:
			if row.HomeTeamID == homeID {
				summary.HomeWins++
			} else {
				summary.AwayWins++
			}
		case ag > hg:
			if row.AwayTeamID == awayID {
				summary.AwayWins++
			} else {
				summary.HomeWins++
			}
		default:
			summary.Draws++
		}
		if i < 10 {
			summary.Matches = append(summary.Matches, h2hMatch{
				Date:      row.UtcKickoff,
				Season:    row.Season,
				Home:      teamIDName{ID: row.HomeTeamID, Name: row.HomeName},
				Away:      teamIDName{ID: row.AwayTeamID, Name: row.AwayName},
				HomeGoals: nullIntPtr(row.HomeGoals),
				AwayGoals: nullIntPtr(row.AwayGoals),
			})
		}
	}
	if summary.Played > 0 {
		summary.AvgGoals = math.Round(float64(totalGoals)/float64(summary.Played)*10) / 10
	}
	return summary
}

func fetchElo(ctx context.Context, st *store.Store, teamID int64) (*float64, error) {
	row, err := st.GetEloRating(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get elo for team %d: %w", teamID, err)
	}
	rating := row.Rating
	return &rating, nil
}

func resultLetter(gf, ga int64) string {
	switch {
	case gf > ga:
		return "W"
	case gf < ga:
		return "L"
	default:
		return "D"
	}
}

func derefInt(n sql.NullInt64) int64 { return n.Int64 }

func nullIntPtr(n sql.NullInt64) *int64 { return nullInt64Ptr(n) }
