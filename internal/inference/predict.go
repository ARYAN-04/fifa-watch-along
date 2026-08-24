package inference

import "math"

func (i *Inference) Predict(state [10]float64) (probs [3]float64, err error) {
	scaled := make([]float64, numFeatures)
	for j, v := range state {
		scaled[j] = (v - i.model.Scaler.Mean[j]) / i.model.Scaler.Scale[j]
	}

	lr := softmax(logits(scaled, i.model.Logistic))

	rf := [3]float64{}
	for _, f := range i.model.Folds {
		forest := forestProba(f.Trees, state[:])
		calibrated := calibrate(forest, f.Sigmoids)
		for k := range rf {
			rf[k] += calibrated[k] / float64(len(i.model.Folds))
		}
	}

	for k := range probs {
		probs[k] = (lr[k] + rf[k]) / 2
	}
	return probs, nil
}

func logits(x []float64, lg logistic) [3]float64 {
	var out [3]float64
	for c := range out {
		sum := lg.Intercept[c]
		for j, v := range x {
			sum += v * lg.Coef[c][j]
		}
		out[c] = sum
	}
	return out
}

func softmax(z [3]float64) [3]float64 {
	max := z[0]
	for _, v := range z {
		if v > max {
			max = v
		}
	}
	exp := [3]float64{}
	total := 0.0
	for k, v := range z {
		exp[k] = math.Exp(v - max)
		total += exp[k]
	}
	for k := range exp {
		exp[k] /= total
	}
	return exp
}

func forestProba(trees []treeNode, x0 []float64) [3]float64 {
	var avg [3]float64
	n := float64(len(trees))
	for _, t := range trees {
		node := 0
		for t.Feature[node] != -2 {
			x := float64(float32(x0[t.Feature[node]]))
			if x <= t.Threshold[node] {
				node = t.Left[node]
			} else {
				node = t.Right[node]
			}
		}
		total := 0.0
		for _, v := range t.Value[node] {
			total += v
		}
		for k, v := range t.Value[node] {
			avg[k] += v / total / n
		}
	}
	return avg
}

func calibrate(forest [3]float64, sigmoids []sigmoid) [3]float64 {
	var out [3]float64
	total := 0.0
	for k, s := range sigmoids {
		out[k] = 1.0 / (1.0 + math.Exp(s.A*forest[k]+s.B))
		total += out[k]
	}
	if total == 0 {
		return [3]float64{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0}
	}
	for k := range out {
		out[k] /= total
	}
	return out
}
