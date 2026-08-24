package inference

import (
	"encoding/json"
	"fmt"
	"os"
)

const numFeatures = 10

type scaler struct {
	Mean  []float64 `json:"mean"`
	Scale []float64 `json:"scale"`
}

type logistic struct {
	Coef      [][]float64 `json:"coef"`
	Intercept []float64   `json:"intercept"`
}

type treeNode struct {
	Feature   []int       `json:"feature"`
	Threshold []float64   `json:"threshold"`
	Left      []int       `json:"left"`
	Right     []int       `json:"right"`
	Value     [][]float64 `json:"value"`
}

type sigmoid struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type fold struct {
	Trees    []treeNode `json:"trees"`
	Sigmoids []sigmoid  `json:"sigmoids"`
}

type modelFile struct {
	Scaler   scaler   `json:"scaler"`
	Logistic logistic `json:"logistic"`
	Folds    []fold   `json:"folds"`
}

type Inference struct {
	model modelFile
}

func New(modelPath string) (*Inference, error) {
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("inference: load model: %w", err)
	}
	var m modelFile
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("inference: parse model: %w", err)
	}
	if len(m.Scaler.Mean) != numFeatures || len(m.Logistic.Coef) != 3 || len(m.Folds) == 0 {
		return nil, fmt.Errorf("inference: unexpected model structure in %s", modelPath)
	}
	return &Inference{model: m}, nil
}
