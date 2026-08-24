package api

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

func intPtr64(v int64) *int64 {
	return &v
}
