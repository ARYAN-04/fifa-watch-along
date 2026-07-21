from unittest.mock import patch, MagicMock
from api.models import Match, MatchEvent, WinProbabilitySnapshot
from api.services.poller import poll_match

@patch("api.services.poller.requests.get")
@patch("api.services.poller.os.getenv")
def test_poll_match_success(mock_getenv, mock_get, db):
    # Setup mock env key
    mock_getenv.return_value = "dummy-api-key"
    
    # Mock football-data.org match details response
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "status": "IN_PLAY",
        "score": {
            "fullTime": {
                "home": 1,
                "away": 0
            }
        },
        "goals": [
            {
                "minute": 23,
                "type": "REGULAR",
                "team": {"id": 1, "name": "Argentina"},
                "scorer": {"id": 50, "name": "Lionel Messi"},
                "assist": {"id": 5, "name": "Angel Di Maria"}
            }
        ],
        "bookings": [
            {
                "minute": 41,
                "card": "YELLOW_CARD",
                "team": {"id": 2, "name": "Canada"},
                "player": {"id": 15, "name": "Cyle Larin"}
            }
        ]
    }
    mock_get.return_value = mock_response
    
    # Run the poller overridden with conftest db session
    with patch("api.services.poller.SessionLocal", return_value=db):
        poll_match()
        
    # Verify match details were updated
    match = db.query(Match).filter_by(id=100).first()
    assert match.status == "IN_PLAY"
    assert match.home_score == 1
    assert match.away_score == 0
    
    # Verify events list was populated
    events = db.query(MatchEvent).filter_by(match_id=100).all()
    assert len(events) == 2
    
    goal_event = [e for e in events if e.event_type == "GOAL"][0]
    assert goal_event.minute == 23
    assert goal_event.player_name == "Lionel Messi"
    assert goal_event.assist_name == "Angel Di Maria"
    
    card_event = [e for e in events if e.event_type == "YELLOW_CARD"][0]
    assert card_event.minute == 41
    assert card_event.player_name == "Cyle Larin"
    
    # Verify win probability snapshot was generated and stored
    snapshots = db.query(WinProbabilitySnapshot).filter_by(match_id=100).all()
    assert len(snapshots) == 1
    assert snapshots[0].score_diff == 1
    assert snapshots[0].home_win_prob >= 0.0
    assert snapshots[0].draw_prob >= 0.0
    assert snapshots[0].away_win_prob >= 0.0
    
    # Assert probabilities sum to 1.0 (within mathematical tolerance)
    assert abs(snapshots[0].home_win_prob + snapshots[0].draw_prob + snapshots[0].away_win_prob - 1.0) < 1e-4
