from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from api.db import get_db
from api.models import MatchConfig, WinProbabilitySnapshot
from api.schemas import WinProbabilityResponse, CurrentWinProbability, WinProbabilityHistoryItem

router = APIRouter()

@router.get("/api/win-probability", response_model=WinProbabilityResponse)
@router.get("/api/win-probability/", response_model=WinProbabilityResponse)
def get_win_probability(db: Session = Depends(get_db)):
    cfg = db.query(MatchConfig).first()
    if not cfg or not cfg.current_match_id:
        raise HTTPException(status_code=404, detail="No match configured")

    snapshots = db.query(WinProbabilitySnapshot).filter(
        WinProbabilitySnapshot.match_id == cfg.current_match_id
    ).order_by(WinProbabilitySnapshot.minute.asc()).all()

    current_snapshot = snapshots[-1] if snapshots else None
    current_data = None
    if current_snapshot:
        current_data = CurrentWinProbability(
            home_win=current_snapshot.home_win_prob,
            draw=current_snapshot.draw_prob,
            away_win=current_snapshot.away_win_prob
        )

    history_data = [
        WinProbabilityHistoryItem(
            minute=s.minute,
            home_win_prob=s.home_win_prob,
            draw_prob=s.draw_prob,
            away_win_prob=s.away_win_prob,
            score_diff=s.score_diff,
            xg_diff_approx=s.xg_diff_approx
        )
        for s in snapshots
    ]

    return WinProbabilityResponse(
        current=current_data,
        history=history_data
    )
