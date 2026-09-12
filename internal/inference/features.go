package inference

import "math"

const defaultElo = 1500.0

// BuildFeatures mirrors ml/features.py build_game_state_features exactly.
// Feature order: score_diff, minute_norm, time_remaining, xg_diff,
// elo_diff/400, red_card_diff, score_diff*time_remaining, effective_lead,
// lead_in_remaining_time, lead_per_time.
func BuildFeatures(homeGoals, awayGoals, minute, eloHome, eloAway float64) [10]float64 {
	if eloHome == 0 {
		eloHome = defaultElo
	}
	if eloAway == 0 {
		eloAway = defaultElo
	}
	scoreDiff := homeGoals - awayGoals
	minuteNorm := minFloat(minute, 95) / 95.0
	timeRemaining := 1.0 - minuteNorm

	var f [10]float64
	f[0] = scoreDiff
	f[1] = minuteNorm
	f[2] = timeRemaining
	f[3] = 0
	f[4] = (eloHome - eloAway) / 400.0
	f[5] = 0
	f[6] = scoreDiff * timeRemaining
	f[7] = scoreDiff * math.Sqrt(minuteNorm)
	f[8] = scoreDiff * (1.0 - math.Sqrt(timeRemaining))
	f[9] = scoreDiff / (timeRemaining + 0.1)
	return f
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
