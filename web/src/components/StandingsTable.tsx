import { useQuery } from '@tanstack/react-query'
import { Trophy } from 'lucide-react'
import { api, type StandingRow } from '../lib/api'

function FormCell({ row }: { row: StandingRow }) {
  const top4 = row.position <= 4
  return (
    <td
      className={`px-2 py-1.5 text-right tabular-nums ${top4 ? 'text-emerald-400' : 'text-zinc-300'}`}
    >
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
    return (
      <div className="h-64 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900" />
    )
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        Failed to load standings.
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900">
      <div className="flex items-center gap-2 border-b border-zinc-800 px-4 py-3">
        <Trophy className="h-4 w-4 text-zinc-500" />
        <h2 className="text-sm font-semibold uppercase tracking-widest text-zinc-400">
          Standings · {data.season}
        </h2>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-xs uppercase tracking-wider text-zinc-500">
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
            <tr key={row.team.id} className="border-t border-zinc-800/60 hover:bg-zinc-800/40">
              <td className="px-3 py-1.5 tabular-nums text-zinc-500">{row.position}</td>
              <td className="px-2 py-1.5 text-zinc-200">{row.team.name}</td>
              <td className="px-2 py-1.5 text-right tabular-nums text-zinc-300">{row.played}</td>
              <td className="px-2 py-1.5 text-right tabular-nums text-zinc-300">{row.won}</td>
              <td className="px-2 py-1.5 text-right tabular-nums text-zinc-300">{row.drawn}</td>
              <td className="px-2 py-1.5 text-right tabular-nums text-zinc-300">{row.lost}</td>
              <td className="hidden px-2 py-1.5 text-right tabular-nums text-zinc-500 sm:table-cell">
                {row.gf}
              </td>
              <td className="hidden px-2 py-1.5 text-right tabular-nums text-zinc-500 sm:table-cell">
                {row.ga}
              </td>
              <td className="px-2 py-1.5 text-right tabular-nums text-zinc-300">
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
