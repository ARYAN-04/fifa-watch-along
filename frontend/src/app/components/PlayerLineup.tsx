'use client';

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
  const renderColumn = (team: TeamGroup, titleColorClass: string) => (
    <div className="flex-1">
      <h3 className={`font-serif font-bold text-base ${titleColorClass} mb-2`}>
        {team.name}
      </h3>
      <div className="text-[10px] text-muted-brown uppercase tracking-wider mb-2 font-mono">
        Roster rating details
      </div>
      {team.players.length === 0 ? (
        <p className="text-muted-brown text-xs font-mono py-4">No ratings seeded yet</p>
      ) : (
        <ul className="divide-y divide-rule border-t border-b border-rule">
          {team.players.map((player) => (
            <li key={player.id} className="grid grid-cols-[40px_1fr_30px] gap-2 items-center py-2 text-xs font-mono text-ink">
              <span className="text-muted-brown font-bold uppercase">{player.position || 'N/A'}</span>
              <span className="truncate pr-1">{player.name}</span>
              <span className="font-serif font-black text-gold text-right text-base leading-none">{player.overall_rating}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );

  return (
    <div>
      <h2 className="font-serif font-bold text-xl mb-1">Squad Ratings</h2>
      <hr className="border-t-2 border-ink mb-4" />
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-1 gap-8">
        {renderColumn(homeTeam, 'text-navy')}
        {renderColumn(awayTeam, 'text-brick')}
      </div>
    </div>
  );
}
