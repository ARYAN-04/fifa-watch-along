package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

type matchDetail struct {
	ID         int64   `json:"id"`
	External   *string `json:"externalId"`
	Season     string  `json:"season"`
	UtcKickoff string  `json:"utcKickoff"`
	Status     string  `json:"status"`
	Home       teamRef `json:"home"`
	Away       teamRef `json:"away"`
	HomeGoals  *int64  `json:"homeGoals"`
	AwayGoals  *int64  `json:"awayGoals"`
	Minute     *int64  `json:"minute"`
}

type matchEvent struct {
	Minute int64   `json:"minute"`
	Type   string  `json:"type"`
	Side   string  `json:"side"`
	Player *string `json:"player"`
	Detail *string `json:"detail"`
}

type probTriple struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

type winProbSnapshotOut struct {
	Minute int64 `json:"minute"`
	probTriple
}

type winProbabilityResponse struct {
	PreMatch  *probTriple          `json:"preMatch"`
	Snapshots []winProbSnapshotOut `json:"snapshots"`
}

func handleMatchDetail(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseMatchID(w, r)
		if !ok {
			return
		}
		row, err := st.GetMatchByID(r.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("api: match %d: %v", id, err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		detail, err := buildMatchDetail(r.Context(), st, row)
		if err != nil {
			log.Printf("api: match %d detail: %v", id, err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		writeJSON(w, detail)
	}
}

func handleMatchEvents(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseMatchID(w, r)
		if !ok {
			return
		}
		if _, err := st.GetMatchByID(r.Context(), id); err != nil {
			respondMatchLookupErr(w, id, err)
			return
		}
		rows, err := st.GetMatchEvents(r.Context(), id)
		if err != nil {
			log.Printf("api: events for match %d: %v", id, err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		events := make([]matchEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, matchEvent{
				Minute: row.Minute,
				Type:   row.Type,
				Side:   row.Side,
				Player: nullStringPtr(row.Player),
				Detail: nullStringPtr(row.Detail),
			})
		}
		writeJSON(w, map[string][]matchEvent{"events": events})
	}
}

func handleWinProbability(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseMatchID(w, r)
		if !ok {
			return
		}
		if _, err := st.GetMatchByID(r.Context(), id); err != nil {
			respondMatchLookupErr(w, id, err)
			return
		}
		rows, err := st.ListWinProbSnapshotsByMatch(r.Context(), id)
		if err != nil {
			log.Printf("api: snapshots for match %d: %v", id, err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		resp := winProbabilityResponse{Snapshots: make([]winProbSnapshotOut, 0, len(rows))}
		if len(rows) > 0 {
			first := rows[0]
			resp.PreMatch = &probTriple{Home: first.Home, Draw: first.Draw, Away: first.Away}
		}
		for _, row := range rows {
			resp.Snapshots = append(resp.Snapshots, winProbSnapshotOut{
				Minute:     row.Minute,
				probTriple: probTriple{Home: row.Home, Draw: row.Draw, Away: row.Away},
			})
		}
		writeJSON(w, resp)
	}
}

func buildMatchDetail(ctx context.Context, st *store.Store, row db.Match) (matchDetail, error) {
	home, err := st.GetTeamByID(ctx, row.HomeTeamID)
	if err != nil {
		return matchDetail{}, err
	}
	away, err := st.GetTeamByID(ctx, row.AwayTeamID)
	if err != nil {
		return matchDetail{}, err
	}
	return matchDetail{
		ID:         row.ID,
		External:   nullStringPtr(row.ExternalID),
		Season:     row.Season,
		UtcKickoff: row.UtcKickoff,
		Status:     row.Status,
		Home:       teamRef{Name: home.Name},
		Away:       teamRef{Name: away.Name},
		HomeGoals:  nullInt64Ptr(row.HomeGoals),
		AwayGoals:  nullInt64Ptr(row.AwayGoals),
		Minute:     nullInt64Ptr(row.Minute),
	}, nil
}

func parseMatchID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"invalid match id"}`, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func respondMatchLookupErr(w http.ResponseWriter, id int64, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
	default:
		log.Printf("api: match %d lookup: %v", id, err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
	}
}
