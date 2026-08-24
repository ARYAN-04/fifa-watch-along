package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

type teamRef struct {
	Name string `json:"name"`
}

type liveMatch struct {
	ID        int64   `json:"id"`
	External  *string `json:"externalId"`
	Home      teamRef `json:"home"`
	Away      teamRef `json:"away"`
	HomeGoals *int64  `json:"homeGoals"`
	AwayGoals *int64  `json:"awayGoals"`
	Minute    *int64  `json:"minute"`
	Status    string  `json:"status"`
}

type liveScoresResponse struct {
	Matches []liveMatch `json:"matches"`
}

func handleLiveScores(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := buildLiveScores(r.Context(), st)
		if err != nil {
			log.Printf("api: live scores: %v", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		writeJSON(w, resp)
	}
}

func buildLiveScores(ctx context.Context, st *store.Store) (liveScoresResponse, error) {
	rows, err := st.GetLiveMatches(ctx)
	if err != nil {
		return liveScoresResponse{}, err
	}
	matches := make([]liveMatch, 0, len(rows))
	for _, row := range rows {
		lm, err := toLiveMatch(ctx, st, row)
		if err != nil {
			return liveScoresResponse{}, err
		}
		matches = append(matches, lm)
	}
	return liveScoresResponse{Matches: matches}, nil
}

func toLiveMatch(ctx context.Context, st *store.Store, row db.Match) (liveMatch, error) {
	home, err := st.GetTeamByID(ctx, row.HomeTeamID)
	if err != nil {
		return liveMatch{}, err
	}
	away, err := st.GetTeamByID(ctx, row.AwayTeamID)
	if err != nil {
		return liveMatch{}, err
	}
	lm := liveMatch{
		ID:        row.ID,
		External:  nullStringPtr(row.ExternalID),
		Home:      teamRef{Name: home.Name},
		Away:      teamRef{Name: away.Name},
		HomeGoals: nullInt64Ptr(row.HomeGoals),
		AwayGoals: nullInt64Ptr(row.AwayGoals),
		Minute:    nullInt64Ptr(row.Minute),
		Status:    row.Status,
	}
	return lm, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encode response: %v", err)
	}
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func nullInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}
