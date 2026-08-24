package api

import (
	"encoding/json"
	"net/http"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
)

type Deps struct {
	Store   *store.Store
	Predict func([10]float64) ([3]float64, error)
	Mocks   bool
}

func Register(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /api/health", handleHealth)
	if d.Mocks {
		mux.HandleFunc("GET /api/scores/live", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, mockLiveScores())
		})
		return
	}
	mux.HandleFunc("GET /api/scores/live", handleLiveScores(d.Store))
	mux.HandleFunc("GET /api/teams/compare", handleTeamCompare(d.Store))
	mux.HandleFunc("GET /api/matches/{id}", handleMatchDetail(d.Store))
	mux.HandleFunc("GET /api/matches/{id}/events", handleMatchEvents(d.Store))
	mux.HandleFunc("GET /api/matches/{id}/win-probability", handleWinProbability(d.Store))
	mux.HandleFunc("GET /api/leagues/{leagueId}/standings", handleLeagueStandings(d.Store))
	mux.HandleFunc("GET /api/leagues/{leagueId}/fixtures", handleLeagueFixtures(d.Store))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
