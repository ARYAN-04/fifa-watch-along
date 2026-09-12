import { useQuery } from '@tanstack/react-query'
import { api, type StandingRow } from '../lib/api'

function FormCell({ row }: { row: StandingRow }) {
  const top4 = row.position <= 4
  return (
    <td className={`px-2 py-1.5 text-right tabular-nums ${top4 ? 'font-bold text-ink' : ''}`}>
      {row.points}
    </td>
  )
}

export function StandingsTable({ leagueId }: { leagueId: string }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['standings', leagueId],
    queryFn: () => api.getStandings(leagueId),
    staleTime: 5 * 60_000,
  })

  if (isPending) {
    return <div className="h-64 animate-pulse border border-ink bg-paper2/30" />
  }

  if (isError || !data) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        Failed to load standings.
      </div>
    )
  }

  return (
    <div className="border border-ink bg-paper">
      <h2 className="border-b border-ink px-4 py-3 font-serif text-lg font-bold">
        Standings · {data.season}
      </h2>
      <table className="w-full font-mono text-sm">
        <thead>
          <tr className="text-xs uppercase tracking-wider text-muted-brown">
            <th className="px-3 py-2 text-left font-medium">#</th>
            <th className="px-2 py-2 text-left font-medium">Team</th>
            <th className="px-2 py-2 text-right font-medium">P</th>
            <th className="px-2 py-2 text-right font-medium">W</th>
            <th className="px-2 py-2 text-right font-medium">D</th>
            <th className="px-2 py-2 text-right font-medium">L</th>
            <th className="hidden px-2 py-2 text-right font-medium sm:table-cell">GF</th>
            <th className="hidden px-2 py-2 text-right font-medium sm:table-cell">GA</th>
            <th className="px-2 py-2 text-right font-medium">GD</th>
            <th className="px-2 py-2 text-right font-medium">Pts</th>
          </tr>
        </thead>
        <tbody>
          {data.standings.map((row) => (
            <tr key={row.team.id} className="border-t border-rule/60">
              <td className="px-3 py-1.5 tabular-nums text-muted-brown">{row.position}</td>
              <td className="px-2 py-1.5 font-bold">{row.team.name}</td>
              <td className="px-2 py-1.5 text-right tabular-nums">{row.played}</td>
              <td className="px-2 py-1.5 text-right tabular-nums">{row.won}</td>
              <td className="px-2 py-1.5 text-right tabular-nums">{row.drawn}</td>
              <td className="px-2 py-1.5 text-right tabular-nums">{row.lost}</td>
              <td className="hidden px-2 py-1.5 text-right tabular-nums text-muted-brown sm:table-cell">
                {row.gf}
              </td>
              <td className="hidden px-2 py-1.5 text-right tabular-nums text-muted-brown sm:table-cell">
                {row.ga}
              </td>
              <td className="px-2 py-1.5 text-right tabular-nums">
                {row.gd > 0 ? `+${row.gd}` : row.gd}
              </td>
              <FormCell row={row} />
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
