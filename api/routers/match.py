from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session, joinedload
from api.db import get_db
from api.models import MatchConfig, Match
from api.schemas import MatchStateResponse, TeamBase

router = APIRouter()

@router.get("/api/match", response_model=MatchStateResponse)
@router.get("/api/match/", response_model=MatchStateResponse)
def get_match_state(db: Session = Depends(get_db)):
    cfg = db.query(MatchConfig).options(
        joinedload(MatchConfig.current_match).joinedload(Match.home_team),
        joinedload(MatchConfig.current_match).joinedload(Match.away_team)
    ).first()

    if not cfg or not cfg.current_match:
        raise HTTPException(status_code=404, detail="No match configured")

    match = cfg.current_match
    home = match.home_team
    away = match.away_team

    return MatchStateResponse(
        match_id=match.id,
        status=match.status,
        home_team=TeamBase.model_validate(home),
        away_team=TeamBase.model_validate(away),
        home_score=match.home_score,
        away_score=match.away_score,
        kickoff_utc=match.kickoff_utc.isoformat(),
        stage=match.stage,
        venue=match.venue
    )
