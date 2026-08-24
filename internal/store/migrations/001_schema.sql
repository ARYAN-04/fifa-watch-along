CREATE TABLE IF NOT EXISTS competitions (
    id INTEGER PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    country TEXT,
    enabled INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    short_name TEXT,
    country TEXT
);

CREATE TABLE IF NOT EXISTS id_crosswalk (
    team_id INTEGER REFERENCES teams(id),
    comp_code TEXT,
    provider TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    UNIQUE(team_id, provider)
);

CREATE TABLE IF NOT EXISTS matches (
    id INTEGER PRIMARY KEY,
    competition_id INTEGER NOT NULL REFERENCES competitions(id),
    external_id TEXT UNIQUE,
    season TEXT NOT NULL,
    utc_kickoff TEXT NOT NULL,
    status TEXT NOT NULL,
    home_team_id INTEGER NOT NULL REFERENCES teams(id),
    away_team_id INTEGER NOT NULL REFERENCES teams(id),
    home_goals INTEGER,
    away_goals INTEGER,
    minute INTEGER
);

CREATE TABLE IF NOT EXISTS match_events (
    id INTEGER PRIMARY KEY,
    match_id INTEGER NOT NULL REFERENCES matches(id),
    minute INTEGER NOT NULL,
    type TEXT NOT NULL,
    side TEXT NOT NULL,
    player TEXT,
    detail TEXT
);

CREATE INDEX IF NOT EXISTS idx_match_events_match_id ON match_events(match_id);

CREATE TABLE IF NOT EXISTS win_prob_snapshots (
    id INTEGER PRIMARY KEY,
    match_id INTEGER NOT NULL REFERENCES matches(id),
    minute INTEGER NOT NULL,
    home REAL NOT NULL,
    draw REAL NOT NULL,
    away REAL NOT NULL,
    UNIQUE(match_id, minute)
);

CREATE TABLE IF NOT EXISTS elo_ratings (
    team_id INTEGER PRIMARY KEY REFERENCES teams(id),
    rating REAL NOT NULL,
    updated_at TEXT NOT NULL
);
