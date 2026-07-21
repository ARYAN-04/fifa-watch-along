from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from api.db import get_db
from api.models import Team, Player
from api.schemas import TeamPlayersResponse, PlayerItem

router = APIRouter()

@router.get("/api/players/{team_id}", response_model=TeamPlayersResponse)
@router.get("/api/players/{team_id}/", response_model=TeamPlayersResponse)
def get_team_players(team_id: int, db: Session = Depends(get_db)):
    team = db.query(Team).filter(Team.id == team_id).first()
    if not team:
        raise HTTPException(status_code=404, detail="Team not found")

    players_qs = db.query(Player).filter(
        Player.team_id == team_id
    ).order_by(Player.overall_rating.desc()).all()

    player_items = [
        PlayerItem(
            id=p.id,
            name=p.name,
            position=p.position,
            overall_rating=p.overall_rating,
            pace=p.pace,
            shooting=p.shooting,
            passing=p.passing,
            dribbling=p.dribbling,
            defending=p.defending,
            physical=p.physical,
            skill_moves=p.skill_moves,
            weak_foot=p.weak_foot
        )
        for p in players_qs
    ]

    return TeamPlayersResponse(
        team=team.name,
        team_id=team.id,
        players=player_items
    )
