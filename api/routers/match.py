from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session, joinedload
from typing import Optional
from api.db import get_db
from api.models import MatchConfig, Match, MatchEvent, WinProbabilitySnapshot
from api.schemas import MatchStateResponse, TeamBase

router = APIRouter()

@router.get("/api/match", response_model=MatchStateResponse)
@router.get("/api/match/", response_model=MatchStateResponse)
def get_match_state(
    db: Session = Depends(get_db),
    minute: Optional[int] = Query(None, description="Replay minute to get match state at"),
):
    cfg = db.query(MatchConfig).options(
        joinedload(MatchConfig.current_match).joinedload(Match.home_team),
        joinedload(MatchConfig.current_match).joinedload(Match.away_team)
    ).first()

    if not cfg or not cfg.current_match:
        raise HTTPException(status_code=404, detail="No match configured")

    match = cfg.current_match
    home = match.home_team
    away = match.away_team

    if minute is not None:
        goals = db.query(MatchEvent).filter(
            MatchEvent.match_id == match.id,
            MatchEvent.event_type == "GOAL",
            MatchEvent.minute <= minute
        ).all()

        home_score = 0
        away_score = 0
        for g in goals:
            if g.team_id == match.home_team_id:
                home_score += 1
            else:
                away_score += 1

        status = "FINISHED" if minute >= 95 else "IN_PLAY"
        return MatchStateResponse(
            match_id=match.id,
            status=status,
            home_team=TeamBase.model_validate(home),
            away_team=TeamBase.model_validate(away),
            home_score=home_score,
            away_score=away_score,
            kickoff_utc=match.kickoff_utc.isoformat(),
            stage=match.stage,
            venue=match.venue
        )

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
