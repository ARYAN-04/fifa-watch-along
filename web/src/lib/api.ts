export interface TeamRef {
  id?: number
  name: string
}

export interface LiveMatch {
  id: number
  externalId: string
  home: TeamRef
  away: TeamRef
  homeGoals: number
  awayGoals: number
  minute: number | null
  status: string
}

export interface LiveScoresResponse {
  matches: LiveMatch[]
}

export interface HealthResponse {
  status: string
}

export interface StandingRow {
  position: number
  team: { id: number; name: string }
  played: number
  won: number
  drawn: number
  lost: number
  gf: number
  ga: number
  gd: number
  points: number
}

export interface StandingsResponse {
  league: string
  season: string
  standings: StandingRow[]
}

export interface Fixture {
  id: number
  externalId: string
  kickoff: string
  status: string
  home: { id: number; name: string }
  away: { id: number; name: string }
  homeGoals: number | null
  awayGoals: number | null
}

export interface FixturesResponse {
  league: string
  season: string
  fixtures: Fixture[]
}

export interface MatchDetail {
  id: number
  externalId: string
  season: string
  utcKickoff: string
  status: string
  home: TeamRef
  away: TeamRef
  homeGoals: number | null
  awayGoals: number | null
  minute: number | null
}

export interface MatchEvent {
  minute: number
  type: 'GOAL' | 'CARD' | 'SUB' | string
  side: 'HOME' | 'AWAY'
  player: string
  detail: string
}

export interface MatchEventsResponse {
  events: MatchEvent[]
}

export interface Probabilities {
  home: number
  draw: number
  away: number
}

export interface WinProbabilitySnapshot extends Probabilities {
  minute: number
}

export interface WinProbabilityResponse {
  preMatch: Probabilities | null
  snapshots: WinProbabilitySnapshot[]
}

export interface TeamFormEntry {
  result: 'W' | 'D' | 'L'
  opponent: string
  date: string
  score: string
}

export interface CompareTeam {
  id: number
  name: string
  form: TeamFormEntry[]
  elo: number
}

export interface H2HMatch {
  date: string
  season: string
  home: { id: number; name: string }
  away: { id: number; name: string }
  homeGoals: number
  awayGoals: number
}

export interface TeamCompareResponse {
  home: CompareTeam
  away: CompareTeam
  h2h: {
    played: number
    homeWins: number
    awayWins: number
    draws: number
    avgGoals: number
    matches: H2HMatch[]
  }
}

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(path)
  if (!res.ok) {
    throw new Error(`GET ${path}: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  getLiveScores: () => getJson<LiveScoresResponse>('/api/scores/live'),
  getHealth: () => getJson<HealthResponse>('/api/health'),
  getStandings: (leagueId: string) =>
    getJson<StandingsResponse>(`/api/leagues/${leagueId}/standings`),
  getFixtures: (leagueId: string, season?: string) =>
    getJson<FixturesResponse>(
      `/api/leagues/${leagueId}/fixtures${season ? `?season=${encodeURIComponent(season)}` : ''}`,
    ),
  getMatch: (id: number) => getJson<MatchDetail>(`/api/matches/${id}`),
  getMatchEvents: (id: number) => getJson<MatchEventsResponse>(`/api/matches/${id}/events`),
  getWinProbability: (id: number) =>
    getJson<WinProbabilityResponse>(`/api/matches/${id}/win-probability`),
  getTeamCompare: (homeId: number, awayId: number) =>
    getJson<TeamCompareResponse>(`/api/teams/compare?home=${homeId}&away=${awayId}`),
}
