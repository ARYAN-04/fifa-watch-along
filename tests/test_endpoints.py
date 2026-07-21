def test_health(client):
    response = client.get("/api/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}

def test_match_state(client):
    response = client.get("/api/match/")
    assert response.status_code == 200
    data = response.json()
    assert data["match_id"] == 100
    assert data["status"] == "SCHEDULED"
    assert data["home_team"]["name"] == "Argentina"
    assert data["away_team"]["name"] == "Canada"
    assert data["home_score"] == 0
    assert data["away_score"] == 0

def test_win_probability_empty(client):
    response = client.get("/api/win-probability/")
    assert response.status_code == 200
    data = response.json()
    assert data["current"] is None
    assert data["history"] == []

def test_events_empty(client):
    response = client.get("/api/events/")
    assert response.status_code == 200
    data = response.json()
    assert data == []

def test_standings(client):
    response = client.get("/api/standings/")
    assert response.status_code == 200
    data = response.json()
    assert "A" in data
    assert len(data["A"]) == 2
    assert data["A"][0]["team"] == "Argentina"
    assert data["A"][0]["points"] == 3
    assert data["A"][1]["team"] == "Canada"
    assert data["A"][1]["points"] == 0

def test_players(client):
    response = client.get("/api/players/1/")
    assert response.status_code == 200
    data = response.json()
    assert data["team"] == "Argentina"
    assert len(data["players"]) == 1
    assert data["players"][0]["name"] == "Lionel Messi"
    assert data["players"][0]["overall_rating"] == 91

def test_players_not_found(client):
    response = client.get("/api/players/999/")
    assert response.status_code == 404
