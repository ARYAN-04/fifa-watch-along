from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session, joinedload
from typing import Dict, List
from api.db import get_db
from api.models import Standing
from api.schemas import StandingItem

router = APIRouter()

@router.get("/api/standings", response_model=Dict[str, List[StandingItem]])
@router.get("/api/standings/", response_model=Dict[str, List[StandingItem]])
def get_standings(db: Session = Depends(get_db)):
    standings_qs = db.query(Standing).options(
        joinedload(Standing.team)
    ).order_by(Standing.group.asc(), Standing.position.asc()).all()

    groups = {}
    for s in standings_qs:
        g = s.group
        if g not in groups:
            groups[g] = []
        groups[g].append(
            StandingItem(
                position=s.position,
                team=s.team.name,
                team_id=s.team_id,
                played=s.played,
                won=s.won,
                drawn=s.drawn,
                lost=s.lost,
                goals_for=s.goals_for,
                goals_against=s.goals_against,
                points=s.points
            )
        )

    return groups
