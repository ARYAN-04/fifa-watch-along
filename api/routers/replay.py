from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session, joinedload
from api.db import get_db
from api.models import MatchConfig, Match

router = APIRouter()

@router.get("/api/replay/config")
@router.get("/api/replay/config/")
def get_replay_config(db: Session = Depends(get_db)):
    cfg = db.query(MatchConfig).first()
    if not cfg or not cfg.current_match_id:
        raise HTTPException(status_code=404, detail="No match configured")
    match = db.query(Match).options(
        joinedload(Match.home_team),
        joinedload(Match.away_team)
    ).filter(Match.id == cfg.current_match_id).first()
    if not match:
        raise HTTPException(status_code=404, detail="Match not found")
    return {
        "match_id": match.id,
        "home_team": match.home_team.name,
        "away_team": match.away_team.name,
        "home_team_id": match.home_team_id,
        "away_team_id": match.away_team_id,
        "stage": match.stage,
        "venue": match.venue,
        "max_minute": 95,
    }

@router.get("/api/replay/matches")
@router.get("/api/replay/matches/")
def list_replay_matches(db: Session = Depends(get_db)):
    matches = db.query(Match).options(
        joinedload(Match.home_team),
        joinedload(Match.away_team)
    ).order_by(Match.id.desc()).all()
    return [
        {
            "match_id": m.id,
            "home_team": m.home_team.name if m.home_team else "Unknown",
            "away_team": m.away_team.name if m.away_team else "Unknown",
            "stage": m.stage,
            "home_score": m.home_score,
            "away_score": m.away_score,
        }
        for m in matches
    ]

@router.post("/api/replay/switch/{match_id}")
@router.post("/api/replay/switch/{match_id}/")
def switch_replay_match(match_id: int, db: Session = Depends(get_db)):
    match = db.query(Match).filter(Match.id == match_id).first()
    if not match:
        raise HTTPException(status_code=404, detail="Match not found in database")
    cfg = db.query(MatchConfig).first()
    if not cfg:
        cfg = MatchConfig(current_match_id=match_id)
        db.add(cfg)
    else:
        cfg.current_match_id = match_id
    db.commit()
    return {"match_id": match_id, "status": "switched"}
