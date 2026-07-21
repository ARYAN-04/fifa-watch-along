from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session, joinedload
from typing import List, Optional
from api.db import get_db
from api.models import MatchConfig, MatchEvent
from api.schemas import MatchEventResponse

router = APIRouter()

@router.get("/api/events", response_model=List[MatchEventResponse])
@router.get("/api/events/", response_model=List[MatchEventResponse])
def get_match_events(
    db: Session = Depends(get_db),
    minute: Optional[int] = Query(None, description="Replay minute to filter events up to"),
):
    cfg = db.query(MatchConfig).first()
    if not cfg or not cfg.current_match_id:
        raise HTTPException(status_code=404, detail="No match configured")

    events_q = db.query(MatchEvent).options(
        joinedload(MatchEvent.team)
    ).filter(
        MatchEvent.match_id == cfg.current_match_id
    )

    if minute is not None:
        events_q = events_q.filter(MatchEvent.minute <= minute)

    events = events_q.order_by(MatchEvent.minute.asc()).all()

    return [
        MatchEventResponse(
            id=e.id,
            minute=e.minute,
            event_type=e.event_type,
            team=e.team.name if e.team else None,
            team_id=e.team_id if e.team_id else None,
            player_name=e.player_name,
            assist_name=e.assist_name,
            detail=e.detail
        )
        for e in events
    ]
