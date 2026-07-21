from pydantic import BaseModel
from typing import List, Optional, Dict
from datetime import datetime

# Team schemas
class TeamBase(BaseModel):
    id: int
    name: str
    short_name: str
    flag_url: str
    pre_match_elo: float

    class Config:
        from_attributes = True

# Match schemas
class MatchStateResponse(BaseModel):
    match_id: int
    status: str
    home_team: TeamBase
    away_team: TeamBase
    home_score: int
    away_score: int
    kickoff_utc: str
    stage: str
    venue: str

# Win probability schemas
class CurrentWinProbability(BaseModel):
    home_win: float
    draw: float
    away_win: float

class WinProbabilityHistoryItem(BaseModel):
    minute: int
    home_win_prob: float
    draw_prob: float
    away_win_prob: float
    score_diff: int
    xg_diff_approx: float

class WinProbabilityResponse(BaseModel):
    current: Optional[CurrentWinProbability]
    history: List[WinProbabilityHistoryItem]

# Event schemas
class MatchEventResponse(BaseModel):
    id: int
    minute: int
    event_type: str
    team: Optional[str]
    team_id: Optional[int]
    player_name: str
    assist_name: str
    detail: str

# Standings schemas
class StandingItem(BaseModel):
    position: int
    team: str
    team_id: int
    played: int
    won: int
    drawn: int
    lost: int
    goals_for: int
    goals_against: int
    points: int

# Players schemas
class PlayerItem(BaseModel):
    id: int
    name: str
    position: str
    overall_rating: int
    pace: int
    shooting: int
    passing: int
    dribbling: int
    defending: int
    physical: int
    skill_moves: int
    weak_foot: int

class TeamPlayersResponse(BaseModel):
    team: str
    team_id: int
    players: List[PlayerItem]
