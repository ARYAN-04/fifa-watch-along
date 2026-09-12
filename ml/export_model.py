"""Export the trained sklearn win-probability ensemble to JSON for the Go engine.

Output: ml/export/model.json
  - StandardScaler mean_/scale_
  - LogisticRegression coef_ (3x10, multinomial) + intercept_
  - Per CalibratedClassifierCV fold: all 100 RF trees (nodes as flat arrays)
    and the three one-vs-all Platt sigmoids (a_, b_)

Go reproduction:
  lr_p   = softmax(scaler(X) @ coef.T + intercept)
  rf_p_k = mean over folds of normalize(sigmoid(a_k * forest_p_k + b_k))
  proba  = (lr_p + rf_p) / 2
"""

import json
from pathlib import Path

import joblib

MODEL_PATH = Path(__file__).resolve().parent / "win_prob_model.pkl"
OUT_PATH = Path(__file__).resolve().parent / "export" / "model.json"


def export_tree(tree) -> dict:
    return {
        "feature": [int(v) for v in tree.feature],
        "threshold": [float(v) for v in tree.threshold],
        "left": [int(v) for v in tree.children_left],
        "right": [int(v) for v in tree.children_right],
        "value": [[float(x) for x in row[0]] for row in tree.value],
    }


def main() -> None:
    model = joblib.load(MODEL_PATH)

    lr_pipe = model.named_estimators_["lr"]
    scaler = lr_pipe.named_steps["standardscaler"]
    logreg = lr_pipe.named_steps["logisticregression"]
    calibrated_rf = model.named_estimators_["rf"]

    folds = []
    for cc in calibrated_rf.calibrated_classifiers_:
        rf = cc.estimator
        trees = [export_tree(est.tree_) for est in rf.estimators_]
        sigmoids = [
            {"a": float(cal.a_), "b": float(cal.b_)} for cal in cc.calibrators
        ]
        folds.append({"trees": trees, "sigmoids": sigmoids})

    payload = {
        "scaler": {
            "mean": [float(v) for v in scaler.mean_],
            "scale": [float(v) for v in scaler.scale_],
        },
        "logistic": {
            "coef": [[float(v) for v in row] for row in logreg.coef_],
            "intercept": [float(v) for v in logreg.intercept_],
        },
        "folds": folds,
    }

    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(OUT_PATH, "w") as f:
        json.dump(payload, f)

    total_nodes = sum(len(t["feature"]) for fold in folds for t in fold["trees"])
    print(f"Exported {type(model).__name__} to {OUT_PATH} ({OUT_PATH.stat().st_size / 1e6:.2f} MB)")
    print(f"  scaler: {len(payload['scaler']['mean'])} features")
    print(f"  logistic regression: {len(logreg.coef_)} classes x {len(logreg.coef_[0])} features")
    print(f"  calibration: {len(folds)} folds x 100 trees ({total_nodes} nodes), 3 Platt sigmoids each")


if __name__ == "__main__":
    main()
