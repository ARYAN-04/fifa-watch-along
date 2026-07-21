import os
import time
import logging
import requests
from sqlalchemy.orm import Session

from api.db import SessionLocal
from api.models import Team, Player

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Mapping from football-data.org Team ID to SoFIFA Team ID
# This can be populated as mappings are known.
FD_TO_SOFIFA = {
    # Example mappings:
    # 77: 1369,  # Argentina
    # 76: 1387,  # Canada
}

def sync_players(db: Session):
    crset_url = os.getenv("CRSET_URL", "http://localhost:8001")
    logger.info(f"Starting player sync using CRSet at {crset_url}")
    
    teams = db.query(Team).all()
    if not teams:
        logger.warning("No teams found in database to sync players for.")
        return

    for team in teams:
        sofifa_id = FD_TO_SOFIFA.get(team.id)
        if not sofifa_id:
            logger.info(f"Skipping team {team.name} (ID: {team.id}) — no SoFIFA mapping found in FD_TO_SOFIFA.")
            continue
            
        url = f"{crset_url}/players"
        params = {"team_id": sofifa_id}
        
        logger.info(f"Fetching players for {team.name} from CRSet...")
        try:
            resp = requests.get(url, params=params, timeout=15)
            if resp.status_code != 200:
                # Try fallback endpoint format /teams/{id}/players
                fallback_url = f"{crset_url}/teams/{sofifa_id}/players"
                resp = requests.get(fallback_url, timeout=15)
                
            if resp.status_code == 200:
                player_data_list = resp.json()
                # Handle case where response has a nested 'data' key or is directly a list
                if isinstance(player_data_list, dict) and "data" in player_data_list:
                    player_data_list = player_data_list["data"]
                
                if not isinstance(player_data_list, list):
                    logger.error(f"Unexpected response format from CRSet for team {team.name}: {player_data_list}")
                    continue
                
                for p in player_data_list:
                    # Extract positions: handle string or list of strings
                    pos = p.get("positions", [""])
                    position = pos[0] if isinstance(pos, list) and pos else p.get("position", "")
                    
                    player_id = p.get("id")
                    if not player_id:
                        continue
                        
                    player = db.query(Player).filter(Player.id == player_id).first()
                    if not player:
                        player = Player(id=player_id, team_id=team.id)
                        db.add(player)
                    
                    player.name = p.get("name", "Unknown Player")
                    player.position = position
                    player.overall_rating = p.get("overall_rating", p.get("overallRating", 0))
                    player.pace = p.get("pace", 0)
                    player.shooting = p.get("shooting", 0)
                    player.passing = p.get("passing", 0)
                    player.dribbling = p.get("dribbling", 0)
                    player.defending = p.get("defending", 0)
                    player.physical = p.get("physical", p.get("physic", 0))
                    player.skill_moves = p.get("skill_moves", p.get("skillMoves", 0))
                    player.weak_foot = p.get("weak_foot", p.get("weakFoot", 0))
                    
                db.commit()
                logger.info(f"Successfully synced {len(player_data_list)} players for team {team.name}")
            else:
                logger.error(f"Failed to fetch players for {team.name}: status {resp.status_code}")
        except Exception as e:
            logger.error(f"Error fetching players for {team.name}: {e}")
            
        time.sleep(1.0) # Rate limiting friendly pause

if __name__ == "__main__":
    db = SessionLocal()
    try:
        sync_players(db)
    finally:
        db.close()
