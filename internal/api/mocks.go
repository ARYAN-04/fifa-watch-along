package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func mockLiveScores() liveScoresResponse {
	minute := 63
	homeGoals := 2
	awayGoals := 1
	kickoffMinute := 1
	extA, extB := "5000001", "5000002"
	return liveScoresResponse{
		Matches: []liveMatch{
			{
				ID:        900001,
				External:  &extA,
				Home:      teamRef{Name: "Arsenal"},
				Away:      teamRef{Name: "Chelsea"},
				HomeGoals: intPtr64(int64(homeGoals)),
				AwayGoals: intPtr64(int64(awayGoals)),
				Minute:    intPtr64(int64(minute)),
				Status:    "LIVE",
			},
			{
				ID:       900002,
				External: &extB,
				Home:     teamRef{Name: "Liverpool"},
				Away:     teamRef{Name: "Everton"},
				Minute:   intPtr64(int64(kickoffMinute)),
				Status:   "LIVE",
			},
		},
	}
}

var mockTeamNames = map[int64]string{
	1:  "Arsenal",
	2:  "Chelsea",
	3:  "Liverpool",
	4:  "Everton",
	28: "Spain",
	29: "Argentina",
	31: "England",
	32: "France",
}

func mockTeamName(id int64) string {
	if name, ok := mockTeamNames[id]; ok {
		return name
	}
	return fmt.Sprintf("Team %d", id)
}

func mockStandings(league string) standingsResponse {
	if league == "WC2026" {
		return mockWC2026Standings()
	}
	if league != "PL" {
		return standingsResponse{League: league, Season: "", Standings: []standingRow{}}
	}
	rows := []struct {
		id                    int64
		played, won, drawn, lost, gf, ga int64
	}{
		{1, 2, 1, 1, 0, 5, 4},
		{3, 2, 1, 1, 0, 7, 3},
		{4, 2, 0, 1, 1, 4, 5},
		{2, 2, 0, 1, 1, 1, 5},
	}
	out := standingsResponse{League: league, Season: "2025-26", Standings: make([]standingRow, 0, len(rows))}
	for i, r := range rows {
		out.Standings = append(out.Standings, standingRow{
			Position: int64(i + 1),
			Team:     teamIDRef{ID: r.id, Name: mockTeamName(r.id)},
			Played:   r.played,
			Won:      r.won,
			Drawn:    r.drawn,
			Lost:     r.lost,
			GF:       r.gf,
			GA:       r.ga,
			GD:       r.gf - r.ga,
			Points:   r.won*3 + r.drawn,
		})
	}
	return out
}

// mockWC2026Standings mirrors the real WC2026 table derived from the four
// seeded finals (Spain 6pts, England 3, Argentina 3, France 0).
func mockWC2026Standings() standingsResponse {
	rows := []struct {
		id                    int64
		played, won, drawn, lost, gf, ga int64
	}{
		{28, 2, 2, 0, 0, 3, 0},
		{31, 2, 1, 0, 1, 7, 6},
		{29, 2, 1, 0, 1, 2, 2},
		{32, 2, 0, 0, 2, 4, 8},
	}
	out := standingsResponse{League: "WC2026", Season: "2026", Standings: make([]standingRow, 0, len(rows))}
	for i, r := range rows {
		out.Standings = append(out.Standings, standingRow{
			Position: int64(i + 1),
			Team:     teamIDRef{ID: r.id, Name: mockTeamName(r.id)},
			Played:   r.played,
			Won:      r.won,
			Drawn:    r.drawn,
			Lost:     r.lost,
			GF:       r.gf,
			GA:       r.ga,
			GD:       r.gf - r.ga,
			Points:   r.won*3 + r.drawn,
		})
	}
	return out
}

func mockFixtures(league, season string) fixturesResponse {
	if league == "WC2026" {
		return fixturesResponse{
			League: league,
			Season: "2026",
			Fixtures: []fixtureRow{
				{
					ID: 1901, Kickoff: "2026-07-14T19:00:00Z", Status: "FINISHED",
					Home: teamIDRef{ID: 32, Name: "France"}, Away: teamIDRef{ID: 28, Name: "Spain"},
					HomeGoals: intPtr64(0), AwayGoals: intPtr64(2),
				},
				{
					ID: 1902, Kickoff: "2026-07-15T19:00:00Z", Status: "FINISHED",
					Home: teamIDRef{ID: 31, Name: "England"}, Away: teamIDRef{ID: 29, Name: "Argentina"},
					HomeGoals: intPtr64(1), AwayGoals: intPtr64(2),
				},
				{
					ID: 1903, Kickoff: "2026-07-18T21:00:00Z", Status: "FINISHED",
					Home: teamIDRef{ID: 32, Name: "France"}, Away: teamIDRef{ID: 31, Name: "England"},
					HomeGoals: intPtr64(4), AwayGoals: intPtr64(6),
				},
				{
					ID: 1904, Kickoff: "2026-07-19T19:00:00Z", Status: "FINISHED",
					Home: teamIDRef{ID: 28, Name: "Spain"}, Away: teamIDRef{ID: 29, Name: "Argentina"},
					HomeGoals: intPtr64(1), AwayGoals: intPtr64(0),
				},
			},
		}
	}
	if league != "PL" {
		return fixturesResponse{League: league, Season: "", Fixtures: []fixtureRow{}}
	}
	extA := "5000001"
	return fixturesResponse{
		League: league,
		Season: season,
		Fixtures: []fixtureRow{
			{
				ID: 900001, External: &extA, Kickoff: "2026-01-10T15:00:00Z", Status: "FINISHED",
				Home: teamIDRef{ID: 1, Name: "Arsenal"}, Away: teamIDRef{ID: 2, Name: "Chelsea"},
				HomeGoals: intPtr64(2), AwayGoals: intPtr64(1),
			},
			{
				ID: 900002, Kickoff: "2026-01-09T20:00:00Z", Status: "FINISHED",
				Home: teamIDRef{ID: 3, Name: "Liverpool"}, Away: teamIDRef{ID: 4, Name: "Everton"},
				HomeGoals: intPtr64(3), AwayGoals: intPtr64(3),
			},
			{
				ID: 900003, Kickoff: "2026-08-30T15:00:00Z", Status: "SCHEDULED",
				Home: teamIDRef{ID: 4, Name: "Everton"}, Away: teamIDRef{ID: 1, Name: "Arsenal"},
			},
		},
	}
}

func mockMatchDetail(id int64) (matchDetail, bool) {
	switch id {
	case 900001:
		ext := "5000001"
		return matchDetail{
			ID: 900001, External: &ext, Season: "2025-26",
			UtcKickoff: "2026-01-10T15:00:00Z", Status: "FINISHED",
			Home: teamRef{Name: "Arsenal"}, Away: teamRef{Name: "Chelsea"},
			HomeGoals: intPtr64(2), AwayGoals: intPtr64(1), Minute: intPtr64(90),
		}, true
	case 900002:
		return matchDetail{
			ID: 900002, Season: "2025-26",
			UtcKickoff: "2026-08-30T15:00:00Z", Status: "SCHEDULED",
			Home: teamRef{Name: "Liverpool"}, Away: teamRef{Name: "Everton"},
		}, true
	default:
		return mockWC2026Detail(id)
	}
}

// mockWC2026 mirrors the seeded WC2026 finals in football.db (ids 1901-1904,
// imported from the previous version via import_legacy) so replay works in
// demo mode without a database.
func mockWC2026Detail(id int64) (matchDetail, bool) {
	rows := map[int64]matchDetail{
		1901: {
			ID: 1901, Season: "2026", UtcKickoff: "2026-07-14T19:00:00Z", Status: "FINISHED",
			Home: teamRef{Name: "France"}, Away: teamRef{Name: "Spain"},
			HomeGoals: intPtr64(0), AwayGoals: intPtr64(2), Minute: intPtr64(90),
		},
		1902: {
			ID: 1902, Season: "2026", UtcKickoff: "2026-07-15T19:00:00Z", Status: "FINISHED",
			Home: teamRef{Name: "England"}, Away: teamRef{Name: "Argentina"},
			HomeGoals: intPtr64(1), AwayGoals: intPtr64(2), Minute: intPtr64(90),
		},
		1903: {
			ID: 1903, Season: "2026", UtcKickoff: "2026-07-18T21:00:00Z", Status: "FINISHED",
			Home: teamRef{Name: "France"}, Away: teamRef{Name: "England"},
			HomeGoals: intPtr64(4), AwayGoals: intPtr64(6), Minute: intPtr64(93),
		},
		1904: {
			ID: 1904, Season: "2026", UtcKickoff: "2026-07-19T19:00:00Z", Status: "FINISHED",
			Home: teamRef{Name: "Spain"}, Away: teamRef{Name: "Argentina"},
			HomeGoals: intPtr64(1), AwayGoals: intPtr64(0), Minute: intPtr64(90),
		},
	}
	d, ok := rows[id]
	return d, ok
}

func ev(minute int64, typ, side, player, detail string) matchEvent {
	return matchEvent{Minute: minute, Type: typ, Side: side, Player: strPtr(player), Detail: strPtr(detail)}
}

func sn(minute int64, h, d, a float64) winProbSnapshotOut {
	return winProbSnapshotOut{Minute: minute, probTriple: probTriple{Home: h, Draw: d, Away: a}}
}

func mockMatchEvents(id int64) ([]matchEvent, bool) {
	if id == 900002 {
		return []matchEvent{}, true
	}
	if id == 900001 {
		return []matchEvent{
			{Minute: 23, Type: "GOAL", Side: "HOME", Player: strPtr("Saka"), Detail: strPtr("Right foot")},
			{Minute: 55, Type: "GOAL", Side: "AWAY", Player: strPtr("Palmer"), Detail: strPtr("Penalty")},
			{Minute: 63, Type: "GOAL", Side: "HOME", Player: strPtr("Havertz"), Detail: strPtr("Header")},
		}, true
	}
	return mockWC2026Events(id)
}

// mockWC2026Events mirrors the seeded event logs in football.db. 1903 keeps
// the real scorer names from the previous version; the other three were
// synthetically generated at seed time and are mirrored verbatim.
func mockWC2026Events(id int64) ([]matchEvent, bool) {
	switch id {
	case 1901:
		return []matchEvent{
			ev(8, "GOAL", "AWAY", "Player A1", "Goal"),
			ev(15, "CARD", "HOME", "Player HOME-YC1", "Yellow Card"),
			ev(30, "CARD", "AWAY", "Player AWAY-YC2", "Yellow Card"),
			ev(40, "CARD", "HOME", "Player HOME-YC3", "Yellow Card"),
			ev(48, "GOAL", "AWAY", "Player A1", "Goal"),
			ev(50, "CARD", "AWAY", "Player AWAY-YC4", "Yellow Card"),
			ev(55, "SUB", "HOME", "P. HOME-On1 ← P. HOME-Off1", "Substitution"),
			ev(65, "CARD", "HOME", "Player HOME-YC5", "Yellow Card"),
			ev(65, "SUB", "AWAY", "P. AWAY-On2 ← P. AWAY-Off2", "Substitution"),
			ev(70, "SUB", "HOME", "P. HOME-On3 ← P. HOME-Off3", "Substitution"),
			ev(75, "SUB", "AWAY", "P. AWAY-On4 ← P. AWAY-Off4", "Substitution"),
			ev(80, "SUB", "HOME", "P. HOME-On5 ← P. HOME-Off5", "Substitution"),
			ev(85, "SUB", "AWAY", "P. AWAY-On6 ← P. AWAY-Off6", "Substitution"),
		}, true
	case 1902:
		return []matchEvent{
			ev(15, "CARD", "HOME", "Player HOME-YC1", "Yellow Card"),
			ev(30, "CARD", "AWAY", "Player AWAY-YC2", "Yellow Card"),
			ev(40, "CARD", "HOME", "Player HOME-YC3", "Yellow Card"),
			ev(48, "GOAL", "AWAY", "Player A1", "Goal"),
			ev(50, "CARD", "AWAY", "Player AWAY-YC4", "Yellow Card"),
			ev(55, "SUB", "HOME", "P. HOME-On1 ← P. HOME-Off1", "Substitution"),
			ev(55, "GOAL", "AWAY", "Player A2", "Goal"),
			ev(65, "CARD", "HOME", "Player HOME-YC5", "Yellow Card"),
			ev(65, "SUB", "AWAY", "P. AWAY-On2 ← P. AWAY-Off2", "Substitution"),
			ev(68, "GOAL", "HOME", "Player H1", "Goal"),
			ev(70, "SUB", "HOME", "P. HOME-On3 ← P. HOME-Off3", "Substitution"),
			ev(75, "SUB", "AWAY", "P. AWAY-On4 ← P. AWAY-Off4", "Substitution"),
			ev(80, "SUB", "HOME", "P. HOME-On5 ← P. HOME-Off5", "Substitution"),
			ev(85, "SUB", "AWAY", "P. AWAY-On6 ← P. AWAY-Off6", "Substitution"),
		}, true
	case 1903:
		return []matchEvent{
			ev(12, "GOAL", "AWAY", "H. Kane", "Goal"),
			ev(23, "CARD", "HOME", "A. Tchouaméni", "Yellow Card"),
			ev(28, "GOAL", "AWAY", "J. Bellingham", "Goal"),
			ev(37, "GOAL", "AWAY", "B. Saka", "Goal"),
			ev(40, "CARD", "AWAY", "D. Rice", "Yellow Card"),
			ev(47, "GOAL", "AWAY", "H. Kane", "Goal"),
			ev(48, "GOAL", "HOME", "K. Mbappé", "Goal"),
			ev(52, "CARD", "AWAY", "K. Walker", "Yellow Card"),
			ev(55, "GOAL", "AWAY", "D. Rice", "Goal"),
			ev(58, "SUB", "HOME", "R. Kolo Muani ← O. Giroud", "Substitution"),
			ev(62, "GOAL", "HOME", "A. Griezmann", "Goal"),
			ev(65, "SUB", "AWAY", "C. Palmer ← M. Rashford", "Substitution"),
			ev(70, "CARD", "HOME", "J. Koundé", "Yellow Card"),
			ev(75, "GOAL", "HOME", "K. Mbappé", "Goal"),
			ev(78, "SUB", "HOME", "E. Camavinga ← A. Griezmann", "Substitution"),
			ev(80, "SUB", "AWAY", "J. Grealish ← B. Saka", "Substitution"),
			ev(85, "CARD", "HOME", "O. Dembélé", "Yellow Card"),
			ev(85, "SUB", "HOME", "M. Thuram ← K. Mbappé", "Substitution"),
			ev(88, "GOAL", "AWAY", "J. Bellingham", "Goal"),
			ev(90, "SUB", "AWAY", "I. Toney ← H. Kane", "Substitution"),
			ev(93, "GOAL", "HOME", "K. Mbappé", "Goal"),
		}, true
	case 1904:
		return []matchEvent{
			ev(15, "CARD", "HOME", "Player HOME-YC1", "Yellow Card"),
			ev(30, "CARD", "AWAY", "Player AWAY-YC2", "Yellow Card"),
			ev(40, "CARD", "HOME", "Player HOME-YC3", "Yellow Card"),
			ev(50, "CARD", "AWAY", "Player AWAY-YC4", "Yellow Card"),
			ev(55, "SUB", "HOME", "P. HOME-On1 ← P. HOME-Off1", "Substitution"),
			ev(65, "CARD", "HOME", "Player HOME-YC5", "Yellow Card"),
			ev(65, "SUB", "AWAY", "P. AWAY-On2 ← P. AWAY-Off2", "Substitution"),
			ev(70, "SUB", "HOME", "P. HOME-On3 ← P. HOME-Off3", "Substitution"),
			ev(75, "SUB", "AWAY", "P. AWAY-On4 ← P. AWAY-Off4", "Substitution"),
			ev(80, "SUB", "HOME", "P. HOME-On5 ← P. HOME-Off5", "Substitution"),
			ev(82, "GOAL", "HOME", "Player H1", "Goal"),
			ev(85, "SUB", "AWAY", "P. AWAY-On6 ← P. AWAY-Off6", "Substitution"),
		}, true
	default:
		return nil, false
	}
}

func mockWinProbability(id int64) (winProbabilityResponse, bool) {
	if id == 900002 {
		return winProbabilityResponse{Snapshots: []winProbSnapshotOut{}}, true
	}
	if id == 900001 {
		pre := &probTriple{Home: 0.55, Draw: 0.25, Away: 0.20}
		return winProbabilityResponse{
			PreMatch: pre,
			Snapshots: []winProbSnapshotOut{
				sn(1, 0.54, 0.26, 0.20),
				sn(10, 0.56, 0.25, 0.19),
				sn(23, 0.68, 0.20, 0.12),
				sn(30, 0.65, 0.21, 0.14),
				sn(45, 0.62, 0.22, 0.16),
				sn(55, 0.45, 0.28, 0.27),
				sn(63, 0.71, 0.18, 0.11),
				sn(75, 0.78, 0.14, 0.08),
				sn(90, 0.97, 0.02, 0.01),
			},
		}, true
	}
	return mockWC2026WinProbability(id)
}

// mockWC2026WinProbability sketches the shape of the seeded 96-point
// timelines (decided results converging late). Demo data, not the DB rows.
func mockWC2026WinProbability(id int64) (winProbabilityResponse, bool) {
	switch id {
	case 1901: // France 0-2 Spain
		return winProbabilityResponse{
			PreMatch: &probTriple{Home: 0.38, Draw: 0.27, Away: 0.35},
			Snapshots: []winProbSnapshotOut{
				sn(1, 0.38, 0.27, 0.35),
				sn(8, 0.28, 0.27, 0.45),
				sn(20, 0.26, 0.28, 0.46),
				sn(30, 0.24, 0.28, 0.48),
				sn(45, 0.22, 0.27, 0.51),
				sn(48, 0.12, 0.20, 0.68),
				sn(60, 0.09, 0.16, 0.75),
				sn(75, 0.05, 0.10, 0.85),
				sn(90, 0.01, 0.02, 0.97),
			},
		}, true
	case 1902: // England 1-2 Argentina
		return winProbabilityResponse{
			PreMatch: &probTriple{Home: 0.40, Draw: 0.27, Away: 0.33},
			Snapshots: []winProbSnapshotOut{
				sn(1, 0.40, 0.27, 0.33),
				sn(20, 0.39, 0.28, 0.33),
				sn(45, 0.37, 0.28, 0.35),
				sn(48, 0.25, 0.28, 0.47),
				sn(55, 0.14, 0.22, 0.64),
				sn(60, 0.12, 0.20, 0.68),
				sn(68, 0.22, 0.28, 0.50),
				sn(80, 0.15, 0.22, 0.63),
				sn(90, 0.04, 0.06, 0.90),
			},
		}, true
	case 1903: // France 4-6 England
		return winProbabilityResponse{
			PreMatch: &probTriple{Home: 0.42, Draw: 0.26, Away: 0.32},
			Snapshots: []winProbSnapshotOut{
				sn(1, 0.42, 0.26, 0.32),
				sn(12, 0.30, 0.27, 0.43),
				sn(28, 0.18, 0.24, 0.58),
				sn(37, 0.08, 0.17, 0.75),
				sn(47, 0.03, 0.08, 0.89),
				sn(48, 0.06, 0.12, 0.82),
				sn(55, 0.04, 0.09, 0.87),
				sn(62, 0.07, 0.13, 0.80),
				sn(75, 0.10, 0.16, 0.74),
				sn(88, 0.04, 0.08, 0.88),
				sn(93, 0.03, 0.05, 0.92),
			},
		}, true
	case 1904: // Spain 1-0 Argentina
		return winProbabilityResponse{
			PreMatch: &probTriple{Home: 0.45, Draw: 0.28, Away: 0.27},
			Snapshots: []winProbSnapshotOut{
				sn(1, 0.45, 0.28, 0.27),
				sn(20, 0.44, 0.29, 0.27),
				sn(45, 0.42, 0.30, 0.28),
				sn(60, 0.40, 0.31, 0.29),
				sn(75, 0.38, 0.32, 0.30),
				sn(82, 0.62, 0.24, 0.14),
				sn(90, 0.88, 0.08, 0.04),
			},
		}, true
	default:
		return winProbabilityResponse{}, false
	}
}

func mockCompare(homeID, awayID int64) compareResponse {
	eloHome, eloAway := 1786.0, 1621.0
	return compareResponse{
		Home: compareTeam{ID: homeID, Name: mockTeamName(homeID), Elo: &eloHome, Form: []formEntry{
			{Result: "W", Opponent: mockTeamName(awayID), Date: "2026-01-10T15:00:00Z", Score: "2-1"},
			{Result: "D", Opponent: "Liverpool", Date: "2026-01-03T17:30:00Z", Score: "2-2"},
		}},
		Away: compareTeam{ID: awayID, Name: mockTeamName(awayID), Elo: &eloAway, Form: []formEntry{
			{Result: "L", Opponent: mockTeamName(homeID), Date: "2026-01-10T15:00:00Z", Score: "1-2"},
			{Result: "D", Opponent: "Everton", Date: "2026-01-04T14:00:00Z", Score: "1-1"},
		}},
		H2H: h2hSummary{
			Played: 2, HomeWins: 1, AwayWins: 0, Draws: 1, AvgGoals: 3,
			Matches: []h2hMatch{
				{
					Date: "2026-01-10T15:00:00Z", Season: "2025-26",
					Home:      teamIDName{ID: homeID, Name: mockTeamName(homeID)},
					Away:      teamIDName{ID: awayID, Name: mockTeamName(awayID)},
					HomeGoals: intPtr64(2), AwayGoals: intPtr64(1),
				},
				{
					Date: "2025-08-30T15:00:00Z", Season: "2025-26",
					Home:      teamIDName{ID: awayID, Name: mockTeamName(awayID)},
					Away:      teamIDName{ID: homeID, Name: mockTeamName(homeID)},
					HomeGoals: intPtr64(1), AwayGoals: intPtr64(1),
				},
			},
		},
	}
}

func handleMockStandings(w http.ResponseWriter, r *http.Request) {
	league := strings.ToUpper(r.PathValue("leagueId"))
	if league == "" {
		league = "PL"
	}
	writeJSON(w, mockStandings(league))
}

func handleMockFixtures(w http.ResponseWriter, r *http.Request) {
	league := strings.ToUpper(r.PathValue("leagueId"))
	if league == "" {
		league = "PL"
	}
	season := r.URL.Query().Get("season")
	if season == "" {
		season = "2025-26"
	}
	writeJSON(w, mockFixtures(league, season))
}

func handleMockMatch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	detail, found := mockMatchDetail(id)
	if !found {
		http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, detail)
}

func handleMockMatchEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	events, found := mockMatchEvents(id)
	if !found {
		http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, map[string][]matchEvent{"events": events})
}

func handleMockWinProbability(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	resp, found := mockWinProbability(id)
	if !found {
		http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, resp)
}

func handleMockCompare(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, mockCompare(homeID, awayID))
}

func intPtr64(v int64) *int64 {
	return &v
}

func strPtr(s string) *string {
	return &s
}
