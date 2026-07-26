import os
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from dotenv import load_dotenv

from api.routers import health, match, win_probability, events, standings, players, replay
from api.services.poller import poll_match

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

load_dotenv()

# Setup scheduler
scheduler = BackgroundScheduler()

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Start APScheduler
    interval = int(os.getenv("POLL_INTERVAL_SECONDS", "60"))
    logger.info(f"Starting background poller scheduler with interval {interval}s")
    scheduler.add_job(
        poll_match,
        trigger="interval",
        seconds=interval,
        id="poll_match",
        replace_existing=True,
        max_instances=1
    )
    scheduler.start()
    
    yield
    
    # Shutdown: Stop scheduler
    logger.info("Stopping background poller scheduler")
    scheduler.shutdown()

app = FastAPI(
    title="FIFA World Cup 2026 Watch-Along API",
    description="FastAPI backend for the FIFA World Cup 2026 Watch-Along Dashboard",
    version="1.0.0",
    lifespan=lifespan
)

# Setup Database Tables & Admin
from api.db import engine
from api.models import Base
from api.admin import setup_admin

Base.metadata.create_all(bind=engine)
setup_admin(app, engine)

# Include Routers
app.include_router(health.router)
app.include_router(match.router)
app.include_router(win_probability.router)
app.include_router(events.router)
app.include_router(standings.router)
app.include_router(players.router)
app.include_router(replay.router)
