'use client';

import { useState, useEffect } from 'react';
import { Activity, Clock } from 'lucide-react';
import WinProbChart from './WinProbChart';
import EventTicker from './EventTicker';
import MatchInfo from './MatchInfo';
import PlayerLineup from './PlayerLineup';

interface TeamInfo {
  id: number;
  name: string;
  short_name: string;
  flag_url: string;
  pre_match_elo: number;
}

interface MatchData {
  match_id: number;
  status: string;
  home_team: TeamInfo;
  away_team: TeamInfo;
  home_score: number;
  away_score: number;
  stage: string;
  venue: string;
}

interface WinProbSnapshot {
  minute: number;
  home_win_prob: number;
  draw_prob: number;
  away_win_prob: number;
}

interface MatchEvent {
  id: number;
  minute: number;
  event_type: string;
  team: string | null;
  team_id: number | null;
  player_name: string;
}

interface PlayerData {
  id: number;
  name: string;
  position: string;
  overall_rating: number;
}

type ViewState =
  | { type: 'loading' }
  | { type: 'error'; message: string }
  | { type: 'no_match' }
  | { type: 'loaded'; match: MatchData; winProb: WinProbSnapshot[]; events: MatchEvent[]; homePlayers: PlayerData[]; awayPlayers: PlayerData[] };

export default function Dashboard() {
  const [isClient, setIsClient] = useState(false);
  const [state, setState] = useState<ViewState>({ type: 'loading' });

  useEffect(() => {
    setIsClient(true);
  }, []);

  useEffect(() => {
    const fetchAll = async () => {
      setState({ type: 'loading' });

      try {
        const [matchRes, winProbRes, eventsRes] = await Promise.all([
          fetch('/api/match/'),
          fetch('/api/win-probability/'),
          fetch('/api/events/'),
        ]);

        if (matchRes.status === 404) {
          setState({ type: 'no_match' });
          return;
        }

        if (!matchRes.ok || !winProbRes.ok || !eventsRes.ok) {
          setState({ type: 'error', message: 'Failed to load match data from backend.' });
          return;
        }

        const matchData: MatchData = await matchRes.json();
        const winProbData = await winProbRes.json();
        const eventsData: MatchEvent[] = await eventsRes.json();

        const [homeRes, awayRes] = await Promise.all([
          fetch(`/api/players/${matchData.home_team.id}/`),
          fetch(`/api/players/${matchData.away_team.id}/`),
        ]);

        const homePlayers: PlayerData[] = homeRes.ok
          ? (await homeRes.json()).players.slice(0, 11)
          : [];

        const awayPlayers: PlayerData[] = awayRes.ok
          ? (await awayRes.json()).players.slice(0, 11)
          : [];

        setState({
          type: 'loaded',
          match: matchData,
          winProb: winProbData.history ?? [],
          events: eventsData,
          homePlayers,
          awayPlayers,
        });
      } catch {
        setState({ type: 'error', message: 'Failed to connect to backend.' });
      }
    };

    fetchAll();
    const interval = setInterval(fetchAll, 30_000);
    return () => clearInterval(interval);
  }, []);

  if (!isClient) return null;

  if (state.type === 'loading') {
    return (
      <div className="min-h-screen bg-zinc-950 text-zinc-100 p-4 md:p-8 font-sans flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Activity className="w-6 h-6 text-rose-500 animate-pulse" />
          <p className="text-zinc-400 text-sm">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  if (state.type === 'error') {
    return (
      <div className="min-h-screen bg-zinc-950 text-zinc-100 p-4 md:p-8 font-sans flex items-center justify-center">
        <div className="text-center">
          <p className="text-rose-500 text-lg font-medium mb-2">Connection Error</p>
          <p className="text-zinc-400 text-sm">{state.message}</p>
          <p className="text-zinc-600 text-xs mt-4">Make sure the Django backend is running on the configured BACKEND_URL</p>
        </div>
      </div>
    );
  }

  if (state.type === 'no_match') {
    return (
      <div className="min-h-screen bg-zinc-950 text-zinc-100 p-4 md:p-8 font-sans flex items-center justify-center">
        <div className="text-center">
          <p className="text-zinc-400 text-lg font-medium mb-2">No Match Configured</p>
          <p className="text-zinc-500 text-sm">Set a current match in the Django admin panel to get started.</p>
        </div>
      </div>
    );
  }

  const { match, winProb, events, homePlayers, awayPlayers } = state;

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 p-4 md:p-8 font-sans selection:bg-rose-500/30">
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">World Cup 2026 Live</h1>
          <p className="text-zinc-400 text-sm mt-1">Live win probability &amp; match events</p>
        </div>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-sm">
          <div className="mb-6 flex justify-between items-end">
            <div>
              <h2 className="text-lg font-medium">Win Probability</h2>
              <p className="text-zinc-400 text-sm mt-1">Live ML inference from game state</p>
            </div>
            <div className="flex gap-4 text-sm font-medium">
              <span className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-rose-500" />
                {' '}{match.home_team.short_name}
              </span>
              <span className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-zinc-600" />
                {' '}{match.away_team.short_name}
              </span>
            </div>
          </div>
          <WinProbChart
            history={winProb}
            homeShortName={match.home_team.short_name}
            awayShortName={match.away_team.short_name}
          />
        </div>

        <div className="lg:col-span-1 bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-sm flex flex-col">
          <div className="mb-6 flex justify-between items-center">
            <h2 className="text-lg font-medium">Match Events</h2>
            <Clock className="w-4 h-4 text-zinc-400" />
          </div>
          <EventTicker events={events} />
        </div>

        <div className="lg:col-span-1 bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-sm">
          <MatchInfo match={match} />
        </div>

        <div className="lg:col-span-2 bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-sm">
          <PlayerLineup
            homeTeam={{ name: match.home_team.name, short_name: match.home_team.short_name, players: homePlayers }}
            awayTeam={{ name: match.away_team.name, short_name: match.away_team.short_name, players: awayPlayers }}
          />
        </div>
      </div>
    </div>
  );
}
