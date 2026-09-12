package api

import (
	"log"
	"net/http"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
)

type replayMatchRow struct {
	ID        int64     `json:"id"`
	League    string    `json:"league"`
	Season    string    `json:"season"`
	Kickoff   string    `json:"kickoff"`
	Status    string    `json:"status"`
	Home      teamIDRef `json:"home"`
	Away      teamIDRef `json:"away"`
	HomeGoals *int64    `json:"homeGoals"`
	AwayGoals *int64    `json:"awayGoals"`
	Snapshots int64     `json:"snapshots"`
}

type replayMatchesResponse struct {
	Matches []replayMatchRow `json:"matches"`
}

func handleReplayMatches(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := st.ListReplayableMatches(r.Context())
		if err != nil {
			log.Printf("api: replay matches: %v", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		out := replayMatchesResponse{Matches: make([]replayMatchRow, 0, len(rows))}
		for _, row := range rows {
			out.Matches = append(out.Matches, replayMatchRow{
				ID:        row.ID,
				League:    row.League,
				Season:    row.Season,
				Kickoff:   row.Kickoff,
				Status:    row.Status,
				Home:      teamIDRef{ID: row.HomeID, Name: row.HomeName},
				Away:      teamIDRef{ID: row.AwayID, Name: row.AwayName},
				HomeGoals: nullInt64Ptr(row.HomeGoals),
				AwayGoals: nullInt64Ptr(row.AwayGoals),
				Snapshots: row.Snapshots,
			})
		}
		writeJSON(w, out)
	}
}

func handleMockReplayMatches(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, replayMatchesResponse{Matches: []replayMatchRow{
		{
			ID: 900001, League: "PL", Season: "2025-26",
			Kickoff: "2026-01-10T15:00:00Z", Status: "FINISHED",
			Home: teamIDRef{ID: 1, Name: "Arsenal"}, Away: teamIDRef{ID: 2, Name: "Chelsea"},
			HomeGoals: intPtr64(2), AwayGoals: intPtr64(1), Snapshots: 9,
		},
		{
			ID: 1904, League: "WC2026", Season: "2026",
			Kickoff: "2026-07-19T19:00:00Z", Status: "FINISHED",
			Home: teamIDRef{ID: 28, Name: "Spain"}, Away: teamIDRef{ID: 29, Name: "Argentina"},
			HomeGoals: intPtr64(1), AwayGoals: intPtr64(0), Snapshots: 96,
		},
		{
			ID: 1903, League: "WC2026", Season: "2026",
			Kickoff: "2026-07-18T21:00:00Z", Status: "FINISHED",
			Home: teamIDRef{ID: 32, Name: "France"}, Away: teamIDRef{ID: 31, Name: "England"},
			HomeGoals: intPtr64(4), AwayGoals: intPtr64(6), Snapshots: 96,
		},
		{
			ID: 1902, League: "WC2026", Season: "2026",
			Kickoff: "2026-07-15T19:00:00Z", Status: "FINISHED",
			Home: teamIDRef{ID: 31, Name: "England"}, Away: teamIDRef{ID: 29, Name: "Argentina"},
			HomeGoals: intPtr64(1), AwayGoals: intPtr64(2), Snapshots: 96,
		},
		{
			ID: 1901, League: "WC2026", Season: "2026",
			Kickoff: "2026-07-14T19:00:00Z", Status: "FINISHED",
			Home: teamIDRef{ID: 32, Name: "France"}, Away: teamIDRef{ID: 28, Name: "Spain"},
			HomeGoals: intPtr64(0), AwayGoals: intPtr64(2), Snapshots: 96,
		},
	}})
}
