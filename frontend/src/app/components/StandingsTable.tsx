'use client';

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

interface Props {
  standings: Record<string, StandingItem[]>;
}

export default function StandingsTable({ standings }: Props) {
  const groups = Object.keys(standings).sort();

  if (groups.length === 0) {
    return (
      <div className="text-muted-brown text-sm py-4 text-center font-mono">
        No standings data available
      </div>
    );
  }

  return (
    <div>
      <h2 className="font-serif font-bold text-xl mb-1">Group Standings</h2>
      <hr className="border-t-2 border-ink mb-6" />
      
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8">
        {groups.map((group) => (
          <div key={group} className="bg-paper2/30 rounded-lg p-4 border border-rule">
            <h3 className="font-serif font-bold text-base text-brick mb-2">Group {group}</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-[11px] font-mono text-left text-ink border-collapse">
                <thead>
                  <tr className="border-b-2 border-ink text-[9px] uppercase tracking-wider text-muted-brown font-bold">
                    <th className="py-2 pr-2">Pos</th>
                    <th className="py-2 pr-4">Team</th>
                    <th className="py-2 text-center px-1">P</th>
                    <th className="py-2 text-center px-1">GD</th>
                    <th className="py-2 text-right pl-2">Pts</th>
                  </tr>
                </thead>
                <tbody>
                  {standings[group].map((s) => {
                    const gd = s.goals_for - s.goals_against;
                    const gdStr = gd > 0 ? `+${gd}` : `${gd}`;
                    return (
                      <tr key={s.team_id} className="border-b border-dotted border-rule last:border-0 hover:bg-paper2/50">
                        <td className="py-2 pr-2 font-bold">{s.position}</td>
                        <td className="py-2 pr-4 font-sans font-semibold text-ink truncate max-w-[110px]">{s.team}</td>
                        <td className="py-2 text-center px-1">{s.played}</td>
                        <td className="py-2 text-center px-1 text-muted-brown">{gdStr}</td>
                        <td className="py-2 text-right pl-2 font-bold text-ink">{s.points}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
