import os
import time
import logging
import requests
from datetime import datetime, timezone

from api.db import SessionLocal
from api.models import MatchConfig, Match, MatchEvent, WinProbabilitySnapshot
from api.services.inference import predict_win_probability

logger = logging.getLogger(__name__)

def poll_match():
    """
    Background job that polls the current match from football-data.org,
    updates match state, processes events, and runs ML inference on score change.
    """
    db = SessionLocal()
    try:
        # 1. Read MatchConfig
        cfg = db.query(MatchConfig).first()
        if not cfg or not cfg.current_match_id:
            logger.info("No active match configured in MatchConfig. Skipping poll.")
            return

        # Load match details with ELOs
        match = db.query(Match).filter(Match.id == cfg.current_match_id).first()
        if not match:
            logger.warning(f"Configured match ID {cfg.current_match_id} not found in database.")
            return

        api_key = os.getenv("FOOTBALL_DATA_API_KEY")
        if not api_key:
            logger.warning("FOOTBALL_DATA_API_KEY is not set in .env. Skipping external poll.")
            return

        headers = {"X-Auth-Token": api_key}
        url = f"https://api.football-data.org/v4/matches/{match.id}"

        # 2. Call external API with retry on 429
        resp = None
        for attempt in range(3):
            try:
                resp = requests.get(url, headers=headers, timeout=10)
                if resp.status_code == 200:
                    break
                elif resp.status_code == 429:
                    retry_after = int(resp.headers.get("Retry-After", 10))
                    logger.warning(f"Rate limited (429). Retrying after {retry_after}s...")
                    time.sleep(retry_after)
                else:
                    logger.error(f"API returned status {resp.status_code} — {resp.text[:200]}")
                    time.sleep(2)
            except Exception as e:
                logger.error(f"HTTP request error: {e}")
                time.sleep(2)

        if not resp or resp.status_code != 200:
            logger.error("Could not retrieve match details from football-data.org.")
            return

        match_data = resp.json()
        status = match_data.get("status", "SCHEDULED")
        score_data = match_data.get("score", {})
        full_time = score_data.get("fullTime", {})
        
        home_score = full_time.get("home", 0) if full_time.get("home") is not None else 0
        away_score = full_time.get("away", 0) if full_time.get("away") is not None else 0

        goals = match_data.get("goals", [])
        bookings = match_data.get("bookings", [])

        # 3. Diff and update Match record
        score_changed = (match.home_score != home_score) or (match.away_score != away_score)
        
        match.status = status
        match.home_score = home_score
        match.away_score = away_score
        db.commit()

        # 4. Process goals
        new_events = 0
        for g in goals:
            minute_event = g.get("minute", 0)
            team_info = g.get("team", {})
            team_id = team_info.get("id")
            scorer = g.get("scorer", {})
            player_name = scorer.get("name", "Unknown")
            assist = g.get("assist", {})
            assist_name = assist.get("name", "") if assist else ""

            # Check if event already exists
            existing = db.query(MatchEvent).filter_by(
                match_id=match.id,
                minute=minute_event,
                event_type="GOAL",
                player_name=player_name
            ).first()

            if not existing:
                event = MatchEvent(
                    match_id=match.id,
                    minute=minute_event,
                    event_type="GOAL",
                    team_id=team_id,
                    player_name=player_name,
                    assist_name=assist_name,
                    detail="Goal"
                )
                db.add(event)
                new_events += 1

        # 5. Process bookings
        red_card_types = {"RED_CARD", "DOUBLE_YELLOW", "RED"}
        for b in bookings:
            minute_event = b.get("minute", 0)
            team_info = b.get("team", {})
            team_id = team_info.get("id")
            player = b.get("player", {})
            player_name = player.get("name", "Unknown")
            card_type = b.get("card", "YELLOW_CARD")

            existing = db.query(MatchEvent).filter_by(
                match_id=match.id,
                minute=minute_event,
                event_type=card_type,
                player_name=player_name
            ).first()

            if not existing:
                event = MatchEvent(
                    match_id=match.id,
                    minute=minute_event,
                    event_type=card_type,
                    team_id=team_id,
                    player_name=player_name,
                    assist_name="",
                    detail=card_type.replace("_", " ").title()
                )
                db.add(event)
                new_events += 1

        if new_events > 0:
            db.commit()

        # 6. Determine current play minute
        if status == "FINISHED":
            current_minute = 90
        elif status == "PAUSED":
            current_minute = 45
        elif status == "IN_PLAY":
            # calculate elapsed minute since kickoff
            if match.kickoff_utc:
                # Parse kickoff time ensuring it's timezone-naive for comparison with utcnow
                kickoff = match.kickoff_utc.replace(tzinfo=None)
                elapsed = (datetime.utcnow() - kickoff).total_seconds() / 60.0
                current_minute = max(1, min(int(elapsed), 95))
            else:
                current_minute = 1
        else:
            current_minute = 0

        # 7. Write WinProbabilitySnapshot if appropriate
        last_snapshot = db.query(WinProbabilitySnapshot).filter_by(
            match_id=match.id
        ).order_by(WinProbabilitySnapshot.minute.desc()).first()

        should_write_snapshot = False
        if not last_snapshot:
            should_write_snapshot = True
        elif score_changed:
            should_write_snapshot = True
        elif status == "IN_PLAY" and current_minute > last_snapshot.minute:
            should_write_snapshot = True

        if should_write_snapshot:
            home_elo = match.home_team.pre_match_elo
            away_elo = match.away_team.pre_match_elo
            pre_match_elo_diff = home_elo - away_elo

            # Count red cards currently in DB
            home_reds = db.query(MatchEvent).filter(
                MatchEvent.match_id == match.id,
                MatchEvent.event_type.in_(red_card_types),
                MatchEvent.team_id == match.home_team_id
            ).count()
            away_reds = db.query(MatchEvent).filter(
                MatchEvent.match_id == match.id,
                MatchEvent.event_type.in_(red_card_types),
                MatchEvent.team_id == match.away_team_id
            ).count()
            red_card_diff = home_reds - away_reds

            score_diff = home_score - away_score
            
            # Approximate xG: each goal is ~0.75 xG, adjusted by ELO
            xg_diff_approx = float(score_diff) * 0.75 + (pre_match_elo_diff / 1000.0)

            probs = predict_win_probability(
                score_diff=score_diff,
                minute=current_minute,
                xg_diff=xg_diff_approx,
                pre_match_elo_diff=pre_match_elo_diff,
                red_card_diff=red_card_diff
            )

            snapshot = WinProbabilitySnapshot(
                match_id=match.id,
                minute=current_minute,
                home_win_prob=probs["home_win"],
                draw_prob=probs["draw"],
                away_win_prob=probs["away_win"],
                score_diff=score_diff,
                xg_diff_approx=xg_diff_approx,
                created_at=datetime.utcnow()
            )
            db.add(snapshot)
            db.commit()
            logger.info(f"Logged win probability snapshot for match {match.id} at minute {current_minute}")

    except Exception as e:
        logger.error(f"Error in poll_match: {e}", exc_info=True)
    finally:
        db.close()
