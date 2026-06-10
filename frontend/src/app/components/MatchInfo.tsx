'use client';

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

interface Props {
  match: MatchData;
}

export default function MatchInfo({ match }: Props) {
  const { home_team, away_team, home_score, away_score } = match;

  return (
    <div>
      <h2 className="text-lg font-medium mb-6">Current Match</h2>
      <div className="flex items-center justify-between">
        <div className="flex flex-col items-center gap-3 flex-1">
          <div className="w-16 h-16 rounded-full bg-zinc-800 flex items-center justify-center border-2 border-rose-500/30">
            <span className="text-xl font-bold">{home_team.short_name}</span>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-rose-500">{home_score}</div>
            <div className="text-xs text-zinc-500 mt-2 font-mono">
              ELO: {home_team.pre_match_elo}
            </div>
          </div>
        </div>

        <div className="text-zinc-600 font-mono text-sm px-4">VS</div>

        <div className="flex flex-col items-center gap-3 flex-1">
          <div className="w-16 h-16 rounded-full bg-zinc-800 flex items-center justify-center border-2 border-zinc-700">
            <span className="text-xl font-bold text-zinc-400">{away_team.short_name}</span>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold">{away_score}</div>
            <div className="text-xs text-zinc-500 mt-2 font-mono">
              ELO: {away_team.pre_match_elo}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
