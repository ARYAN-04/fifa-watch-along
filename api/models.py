from datetime import datetime
from typing import List, Optional
from sqlalchemy import (
    Column, Integer, String, Float, DateTime, ForeignKey,
    UniqueConstraint
)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship

class Base(DeclarativeBase):
    pass

class Team(Base):
    __tablename__ = "dashboard_team"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    name: Mapped[str] = mapped_column(String(100))
    short_name: Mapped[str] = mapped_column(String(10), default="")
    flag_url: Mapped[str] = mapped_column(String(200), default="")
    group: Mapped[str] = mapped_column(String(2), default="")
    pre_match_elo: Mapped[float] = mapped_column(Float, default=1500.0)
    fc26_overall: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Relationships
    players: Mapped[List["Player"]] = relationship(
        "Player", back_populates="team", cascade="all, delete-orphan"
    )
    home_matches: Mapped[List["Match"]] = relationship(
        "Match", foreign_keys="[Match.home_team_id]", back_populates="home_team"
    )
    away_matches: Mapped[List["Match"]] = relationship(
        "Match", foreign_keys="[Match.away_team_id]", back_populates="away_team"
    )

class Player(Base):
    __tablename__ = "dashboard_player"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    team_id: Mapped[int] = mapped_column(Integer, ForeignKey("dashboard_team.id"))
    name: Mapped[str] = mapped_column(String(100))
    position: Mapped[str] = mapped_column(String(10), default="")
    overall_rating: Mapped[int] = mapped_column(Integer, default=0)
    pace: Mapped[int] = mapped_column(Integer, default=0)
    shooting: Mapped[int] = mapped_column(Integer, default=0)
    passing: Mapped[int] = mapped_column(Integer, default=0)
    dribbling: Mapped[int] = mapped_column(Integer, default=0)
    defending: Mapped[int] = mapped_column(Integer, default=0)
    physical: Mapped[int] = mapped_column(Integer, default=0)
    skill_moves: Mapped[int] = mapped_column(Integer, default=0)
    weak_foot: Mapped[int] = mapped_column(Integer, default=0)
    nationality: Mapped[str] = mapped_column(String(50), default="")

    # Relationships
    team: Mapped["Team"] = relationship("Team", back_populates="players")

class Match(Base):
    __tablename__ = "dashboard_match"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    home_team_id: Mapped[int] = mapped_column(Integer, ForeignKey("dashboard_team.id"))
    away_team_id: Mapped[int] = mapped_column(Integer, ForeignKey("dashboard_team.id"))
    kickoff_utc: Mapped[datetime] = mapped_column(DateTime)
    stage: Mapped[str] = mapped_column(String(50))
    venue: Mapped[str] = mapped_column(String(100), default="")
    status: Mapped[str] = mapped_column(String(20), default="SCHEDULED")
    home_score: Mapped[int] = mapped_column(Integer, default=0)
    away_score: Mapped[int] = mapped_column(Integer, default=0)

    # Relationships
    home_team: Mapped["Team"] = relationship("Team", foreign_keys=[home_team_id], back_populates="home_matches")
    away_team: Mapped["Team"] = relationship("Team", foreign_keys=[away_team_id], back_populates="away_matches")
    events: Mapped[List["MatchEvent"]] = relationship(
        "MatchEvent", back_populates="match", cascade="all, delete-orphan"
    )
    win_prob_snapshots: Mapped[List["WinProbabilitySnapshot"]] = relationship(
        "WinProbabilitySnapshot", back_populates="match", cascade="all, delete-orphan"
    )

class MatchEvent(Base):
    __tablename__ = "dashboard_matchevent"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    match_id: Mapped[int] = mapped_column(Integer, ForeignKey("dashboard_match.id"))
    minute: Mapped[int] = mapped_column(Integer)
    event_type: Mapped[str] = mapped_column(String(20))
    team_id: Mapped[Optional[int]] = mapped_column(Integer, ForeignKey("dashboard_team.id"), nullable=True)
    player_name: Mapped[str] = mapped_column(String(100))
    assist_name: Mapped[str] = mapped_column(String(100), default="")
    detail: Mapped[str] = mapped_column(String(50), default="")
    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)

    # Relationships
    match: Mapped["Match"] = relationship("Match", back_populates="events")
    team: Mapped[Optional["Team"]] = relationship("Team")

    __table_args__ = (
        UniqueConstraint(
            "match_id", "minute", "event_type", "player_name",
            name="dashboard_matchevent_match_id_minute_event_type_player_name_f0a6fa54_uniq"
        ),
    )

class WinProbabilitySnapshot(Base):
    __tablename__ = "dashboard_winprobabilitysnapshot"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    match_id: Mapped[int] = mapped_column(Integer, ForeignKey("dashboard_match.id"))
    minute: Mapped[int] = mapped_column(Integer)
    home_win_prob: Mapped[float] = mapped_column(Float)
    draw_prob: Mapped[float] = mapped_column(Float)
    away_win_prob: Mapped[float] = mapped_column(Float)
    score_diff: Mapped[int] = mapped_column(Integer)
    xg_diff_approx: Mapped[float] = mapped_column(Float)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)

    # Relationships
    match: Mapped["Match"] = relationship("Match", back_populates="win_prob_snapshots")

class Standing(Base):
    __tablename__ = "dashboard_standing"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    team_id: Mapped[int] = mapped_column(Integer, ForeignKey("dashboard_team.id"))
    group: Mapped[str] = mapped_column(String(2))
    position: Mapped[int] = mapped_column(Integer)
    played: Mapped[int] = mapped_column(Integer, default=0)
    won: Mapped[int] = mapped_column(Integer, default=0)
    drawn: Mapped[int] = mapped_column(Integer, default=0)
    lost: Mapped[int] = mapped_column(Integer, default=0)
    goals_for: Mapped[int] = mapped_column(Integer, default=0)
    goals_against: Mapped[int] = mapped_column(Integer, default=0)
    points: Mapped[int] = mapped_column(Integer, default=0)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    team: Mapped["Team"] = relationship("Team")

    __table_args__ = (
        UniqueConstraint(
            "team_id", "group",
            name="dashboard_standing_team_id_group_0c1e64df_uniq"
        ),
    )

class MatchConfig(Base):
    __tablename__ = "dashboard_matchconfig"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    current_match_id: Mapped[Optional[int]] = mapped_column(Integer, ForeignKey("dashboard_match.id"), nullable=True)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    current_match: Mapped[Optional["Match"]] = relationship("Match")
