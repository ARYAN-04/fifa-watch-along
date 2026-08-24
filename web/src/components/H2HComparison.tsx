import { useQuery } from '@tanstack/react-query'
import { api, type TeamCompareResponse, type TeamFormEntry } from '../lib/api'

const RESULT_STYLES: Record<TeamFormEntry['result'], string> = {
  W: 'bg-emerald-950 text-emerald-400',
  D: 'bg-zinc-800 text-zinc-400',
  L: 'bg-red-950 text-red-400',
}

function FormGuide({ team }: { team: TeamCompareResponse['home'] }) {
  return (
    <div>
      <h4 className="text-xs font-semibold uppercase tracking-widest text-zinc-500">Form</h4>
      <div className="mt-2 space-y-1.5">
        {team.form.length === 0 ? (
          <p className="text-sm text-zinc-500">No recent matches.</p>
        ) : (
          team.form.map((f, i) => (
            <div key={i} className="flex items-center gap-2 text-sm">
              <span
                className={`flex h-5 w-5 items-center justify-center rounded text-xs font-bold ${RESULT_STYLES[f.result]}`}
              >
                {f.result}
              </span>
              <span className="text-zinc-300">{f.opponent}</span>
              <span className="ml-auto tabular-nums text-xs text-zinc-500">{f.score}</span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

function RecordBar({
  homeWins,
  draws,
  awayWins,
}: {
  homeWins: number
  draws: number
  awayWins: number
}) {
  const total = homeWins + draws + awayWins || 1
  const seg = (n: number) => `${(n / total) * 100}%`
  return (
    <div>
      <div className="flex h-3 overflow-hidden rounded-full bg-zinc-800">
        <div style={{ width: seg(homeWins) }} className="bg-emerald-500" />
        <div style={{ width: seg(draws) }} className="bg-zinc-600" />
        <div style={{ width: seg(awayWins) }} className="bg-red-500" />
      </div>
      <div className="mt-2 flex justify-between text-xs tabular-nums text-zinc-400">
        <span>{homeWins} W</span>
        <span>{draws} D</span>
        <span>{awayWins} W</span>
      </div>
    </div>
  )
}

export function H2HComparison({ homeId, awayId }: { homeId: number; awayId: number }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['teamCompare', homeId, awayId],
    queryFn: () => api.getTeamCompare(homeId, awayId),
    staleTime: 10 * 60_000,
  })

  if (isPending) {
    return <div className="h-72 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900" />
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        Failed to load head-to-head data.
      </div>
    )
  }

  const { home, away, h2h } = data

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4 rounded-lg border border-zinc-800 bg-zinc-900 p-6">
        <div>
          <p className="text-xl font-semibold text-white">{home.name}</p>
          <p className="tabular-nums text-sm text-emerald-400">Elo {Math.round(home.elo)}</p>
        </div>
        <p className="text-xs uppercase tracking-widest text-zinc-600">vs</p>
        <div className="text-right">
          <p className="text-xl font-semibold text-white">{away.name}</p>
          <p className="tabular-nums text-sm text-red-400">Elo {Math.round(away.elo)}</p>
        </div>
      </div>

      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-semibold uppercase tracking-widest text-zinc-400">
            Head to Head · {h2h.played} played
          </h3>
          <span className="text-xs tabular-nums text-zinc-500">
            Avg goals {h2h.avgGoals.toFixed(2)}
          </span>
        </div>
        <RecordBar
          homeWins={h2h.homeWins}
          draws={h2h.draws}
          awayWins={h2h.awayWins}
        />

        {h2h.matches.length > 0 && (
          <table className="mt-6 w-full text-sm">
            <thead>
              <tr className="text-xs uppercase tracking-wider text-zinc-500">
                <th className="py-2 text-left font-medium">Date</th>
                <th className="py-2 text-left font-medium">Season</th>
                <th className="py-2 text-left font-medium">Match</th>
                <th className="py-2 text-right font-medium">Score</th>
              </tr>
            </thead>
            <tbody>
              {h2h.matches.map((m, i) => (
                <tr key={i} className="border-t border-zinc-800/60">
                  <td className="py-1.5 tabular-nums text-zinc-500">{m.date}</td>
                  <td className="py-1.5 text-zinc-500">{m.season}</td>
                  <td className="py-1.5 text-zinc-300">
                    {m.home.name} vs {m.away.name}
                  </td>
                  <td className="py-1.5 text-right font-semibold tabular-nums text-white">
                    {m.homeGoals} - {m.awayGoals}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="grid gap-6 sm:grid-cols-2">
        <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6">
          <FormGuide team={home} />
        </div>
        <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6">
          <FormGuide team={away} />
        </div>
      </div>
    </div>
  )
}
