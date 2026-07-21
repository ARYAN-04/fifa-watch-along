import os
import pytest
from datetime import datetime
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from fastapi.testclient import TestClient

from api.main import app
from api.db import get_db
from api.models import Base, Team, Player, Match, MatchConfig, Standing

DB_FILE = "./test.db"
SQLALCHEMY_DATABASE_URL = f"sqlite:///{DB_FILE}"

@pytest.fixture(scope="function")
def db():
    # Clean up old database file if exists
    if os.path.exists(DB_FILE):
        try:
            os.remove(DB_FILE)
        except Exception:
            pass

    engine = create_engine(
        SQLALCHEMY_DATABASE_URL, connect_args={"check_same_thread": False}
    )
    TestingSessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

    # Create tables
    Base.metadata.create_all(bind=engine)
    session = TestingSessionLocal()
    
    # Seed dummy data for tests
    arg = Team(id=1, name="Argentina", short_name="ARG", flag_url="", group="A", pre_match_elo=2100.0)
    can = Team(id=2, name="Canada", short_name="CAN", flag_url="", group="A", pre_match_elo=1700.0)
    
    session.add(arg)
    session.add(can)
    session.commit()
    
    match = Match(
        id=100,
        home_team_id=arg.id,
        away_team_id=can.id,
        kickoff_utc=datetime.utcnow(),
        stage="Group Stage",
        venue="MetLife Stadium",
        status="SCHEDULED",
        home_score=0,
        away_score=0
    )
    session.add(match)
    session.commit()
    
    config = MatchConfig(
        current_match_id=match.id
    )
    session.add(config)
    
    standing_arg = Standing(
        team_id=arg.id,
        group="A",
        position=1,
        played=1,
        won=1,
        drawn=0,
        lost=0,
        goals_for=2,
        goals_against=0,
        points=3
    )
    standing_can = Standing(
        team_id=can.id,
        group="A",
        position=2,
        played=1,
        won=0,
        drawn=0,
        lost=1,
        goals_for=0,
        goals_against=2,
        points=0
    )
    session.add(standing_arg)
    session.add(standing_can)
    
    player_messi = Player(
        id=50,
        team_id=arg.id,
        name="Lionel Messi",
        position="CF",
        overall_rating=91,
        pace=75,
        shooting=92,
        passing=90,
        dribbling=95,
        defending=34,
        physical=65,
        skill_moves=4,
        weak_foot=4,
        nationality="Argentina"
    )
    session.add(player_messi)
    session.commit()
    
    try:
        yield session
    finally:
        session.close()
        # Clean up database file
        if os.path.exists(DB_FILE):
            try:
                os.remove(DB_FILE)
            except Exception:
                pass

@pytest.fixture(scope="function")
def client(db):
    def _get_db_override():
        try:
            yield db
        finally:
            pass
            
    app.dependency_overrides[get_db] = _get_db_override
    with TestClient(app) as test_client:
        yield test_client
    app.dependency_overrides.clear()
