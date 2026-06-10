'use client';

import { Trophy } from 'lucide-react';

interface PlayerData {
  id: number;
  name: string;
  position: string;
  overall_rating: number;
}

interface TeamGroup {
  name: string;
  short_name: string;
  players: PlayerData[];
}

interface Props {
  homeTeam: TeamGroup;
  awayTeam: TeamGroup;
}

export default function PlayerLineup({ homeTeam, awayTeam }: Props) {
  const renderColumn = (team: TeamGroup, accentColor: string) => (
    <div>
      <h3 className={`text-sm font-semibold ${accentColor} mb-4 pb-2 border-b border-zinc-800`}>
        {team.name}
      </h3>
      {team.players.length === 0 ? (
        <p className="text-zinc-500 text-sm">No player data available</p>
      ) : (
        <ul className="space-y-3">
          {team.players.map((player) => (
            <li key={player.id} className="flex items-center justify-between text-sm">
              <span className="text-zinc-300 truncate mr-2">{player.name}</span>
              <span className="text-zinc-600 font-mono text-xs whitespace-nowrap">
                {player.position ? `${player.position} · ` : ''}{player.overall_rating} OVR
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );

  return (
    <div>
      <div className="mb-6 flex justify-between items-center">
        <h2 className="text-lg font-medium">Top Rated Squad</h2>
        <Trophy className="w-4 h-4 text-zinc-400" />
      </div>
      <div className="grid grid-cols-2 gap-8">
        {renderColumn(homeTeam, 'text-rose-500')}
        {renderColumn(awayTeam, 'text-zinc-400')}
      </div>
    </div>
  );
}
