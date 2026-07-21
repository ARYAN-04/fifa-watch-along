'use client';

import { useState, useEffect } from 'react';
import { Activity, Clock } from 'lucide-react';
import WinProbChart from './WinProbChart';
import EventTicker from './EventTicker';
import MatchInfo from './MatchInfo';
import PlayerLineup from './PlayerLineup';
import StandingsTable from './StandingsTable';

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
  assist_name?: string;
}

interface PlayerData {
  id: number;
  name: string;
  position: string;
  overall_rating: number;
}

interface StandingItem {
  position: number;
  team: string;
  team_id: number;
  played: number;
  won: number;
  drawn: number;
  lost: number;
  goals_for: number;
  goals_against: number;
  points: number;
}

type ViewState =
  | { type: 'loading' }
  | { type: 'error'; message: string }
  | { type: 'no_match' }
  | { type: 'loaded'; match: MatchData; winProb: WinProbSnapshot[]; events: MatchEvent[]; homePlayers: PlayerData[]; awayPlayers: PlayerData[]; standings: Record<string, StandingItem[]> };

export default function Dashboard() {
  const [isClient, setIsClient] = useState(false);
  const [state, setState] = useState<ViewState>({ type: 'loading' });

  useEffect(() => {
    setIsClient(true);
  }, []);

  useEffect(() => {
    const fetchAll = async () => {
      try {
        const [matchRes, winProbRes, eventsRes, standingsRes] = await Promise.all([
          fetch('/api/match/'),
          fetch('/api/win-probability/'),
          fetch('/api/events/'),
          fetch('/api/standings/'),
        ]);

        if (matchRes.status === 404) {
          setState({ type: 'no_match' });
          return;
        }

        if (!matchRes.ok || !winProbRes.ok || !eventsRes.ok || !standingsRes.ok) {
          setState({ type: 'error', message: 'Failed to load match data from backend.' });
          return;
        }

        const matchData: MatchData = await matchRes.json();
        const winProbData = await winProbRes.json();
        const eventsData: MatchEvent[] = await eventsRes.json();
        const standingsData: Record<string, StandingItem[]> = await standingsRes.json();

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
          standings: standingsData,
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
      <div className="min-h-screen bg-paper text-ink p-4 md:p-8 flex items-center justify-center font-mono">
        <div className="flex flex-col items-center gap-3">
          <Activity className="w-6 h-6 text-brick animate-pulse" />
          <p className="text-muted-brown text-sm tracking-wider">LOADING MATCHDAY SHEET...</p>
        </div>
      </div>
    );
  }

  if (state.type === 'error') {
    return (
      <div className="min-h-screen bg-paper text-ink p-4 md:p-8 flex items-center justify-center font-mono">
        <div className="text-center max-w-md border border-ink p-8 bg-paper2/30">
          <p className="text-brick text-lg font-bold mb-2">CONNECTION ERROR</p>
          <p className="text-muted-brown text-sm mb-4">{state.message}</p>
          <p className="text-[10px] text-muted-brown border-t border-rule pt-4 uppercase">
            Ensure the FastAPI backend is running at BACKEND_URL
          </p>
        </div>
      </div>
    );
  }

  if (state.type === 'no_match') {
    return (
      <div className="min-h-screen bg-paper text-ink p-4 md:p-8 flex items-center justify-center font-mono">
        <div className="text-center max-w-md border border-ink p-8 bg-paper2/30">
          <p className="text-brick text-lg font-bold mb-2">NO MATCH ACTIVE</p>
          <p className="text-muted-brown text-sm">Please configure an active match ID in the Admin Panel to begin live watch-along program.</p>
        </div>
      </div>
    );
  }

  const { match, winProb, events, homePlayers, awayPlayers, standings } = state;

  return (
    <div className="w-[calc(100%-2rem)] md:w-[calc(100%-4rem)] max-w-[1400px] xl:max-w-[1600px] mx-auto bg-paper border border-ink relative my-8 shadow-sm">
      
      {/* Masthead */}
      <header className="py-6 px-10 border-b-3 border-double border-ink text-center">
        <div className="text-[11px] tracking-[0.25em] uppercase text-brick mb-1.5 font-bold font-mono">Official Watch-Along Programme</div>
        <h1 className="font-serif font-black text-4xl md:text-5xl tracking-tight text-ink uppercase">The Matchday Sheet</h1>
        <div className="text-[11px] text-muted-brown mt-2 tracking-wide font-mono">
          FIFA World Cup 2026 &nbsp;·&nbsp; {match.stage} &nbsp;·&nbsp; <span className="text-ink font-bold">{match.venue || 'MetLife Stadium'}</span>
        </div>
      </header>

      {/* Main & Side Columns */}
      <div className="flex flex-col md:flex-row border-b border-ink">
        
        {/* Main Column */}
        <div className="flex-1 p-6 md:p-10 md:border-r border-dashed border-ink">
          
          {/* Match Score Ticket */}
          <MatchInfo match={match} />

          {/* Win Probability Section */}
          <div className="mt-8">
            <h2 className="font-serif font-bold text-lg text-ink">Win Probability</h2>
            <hr className="border-t-2 border-ink my-2" />
            <WinProbChart
              history={winProb}
              homeShortName={match.home_team.short_name}
              awayShortName={match.away_team.short_name}
            />
          </div>

          {/* Match Log Section */}
          <div className="mt-10">
            <h2 className="font-serif font-bold text-lg text-ink">Match Log</h2>
            <hr className="border-t-2 border-ink my-2" />
            <EventTicker events={events} />
          </div>
        </div>

        {/* Side Column */}
        <div className="w-full md:w-[320px] lg:w-[380px] xl:w-[420px] p-6 md:p-8 bg-paper2/40 flex flex-col gap-8">
          
          {/* Squad Ratings Section */}
          <PlayerLineup
            homeTeam={{ name: match.home_team.name, short_name: match.home_team.short_name, players: homePlayers }}
            awayTeam={{ name: match.away_team.name, short_name: match.away_team.short_name, players: awayPlayers }}
          />
        </div>
      </div>

      {/* Bottom Full-Width Standings Ledger */}
      <div className="p-6 md:p-10 border-b border-ink">
        <StandingsTable standings={standings} />
      </div>

      {/* Footer Strip */}
      <div className="flex justify-between items-center py-4 px-10 text-[9px] tracking-widest uppercase text-muted-brown font-mono font-bold">
        <span>Data: football-data.org / StatsBomb / SoFIFA</span>
        <span>Printed Live · Do Not Remove Stub</span>
      </div>
    </div>
  );
}
