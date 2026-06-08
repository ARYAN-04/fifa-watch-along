"""
Background polling stub — fully implemented in Phase 3.
This must exist so dashboard/apps.py can import start_scheduler without error.
"""
import os
from apscheduler.schedulers.background import BackgroundScheduler

current_state: dict = {
    'match_id': None,
    'status': 'NO_MATCH',
    'home_team': None,
    'away_team': None,
    'home_team_id': None,
    'away_team_id': None,
    'home_score': 0,
    'away_score': 0,
    'minute': 0,
    'events': [],
    'win_probability': {'home_win': 0.40, 'draw': 0.25, 'away_win': 0.35},
    'win_prob_history': [],
    'last_updated': None,
    'pre_match_elo_diff': 0.0,
}


def poll_match():
    pass


_scheduler = None


def start_scheduler():
    global _scheduler
    if _scheduler is not None:
        return
    interval = int(os.getenv('POLL_INTERVAL_SECONDS', 60))
    _scheduler = BackgroundScheduler()
    _scheduler.add_job(
        poll_match,
        trigger='interval',
        seconds=interval,
        id='poll_match',
        replace_existing=True,
        max_instances=1,
    )
    _scheduler.start()
