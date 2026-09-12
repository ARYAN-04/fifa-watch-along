import { useQuery } from '@tanstack/react-query'
import { api, type TeamCompareResponse, type TeamFormEntry } from '../lib/api'

const RESULT_STYLES: Record<TeamFormEntry['result'], string> = {
  W: 'bg-ink text-paper',
  D: 'bg-paper2 text-muted-brown',
  L: 'bg-brick text-paper',
}

function FormGuide({ team }: { team: TeamCompareResponse['home'] }) {
  return (
    <div>
      <h4 className="font-mono text-xs font-bold uppercase tracking-widest text-muted-brown">Form</h4>
      <div className="mt-2 space-y-1.5">
        {team.form.length === 0 ? (
          <p className="font-mono text-sm text-muted-brown">No recent matches.</p>
        ) : (
          team.form.map((f, i) => (
            <div key={i} className="flex items-center gap-2 font-mono text-sm">
              <span
                className={`flex h-5 w-5 items-center justify-center text-xs font-bold ${RESULT_STYLES[f.result]}`}
              >
                {f.result}
              </span>
              <span>{f.opponent}</span>
              <span className="ml-auto text-xs tabular-nums text-muted-brown">{f.score}</span>
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
      <div className="flex h-3 overflow-hidden bg-rule">
        <div style={{ width: seg(homeWins) }} className="bg-navy" />
        <div style={{ width: seg(draws) }} className="bg-muted-brown" />
        <div style={{ width: seg(awayWins) }} className="bg-brick" />
      </div>
      <div className="mt-2 flex justify-between font-mono text-xs tabular-nums text-muted-brown">
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
    return <div className="h-72 animate-pulse border border-ink bg-paper2/30" />
  }

  if (isError || !data) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        Failed to load head-to-head data.
      </div>
    )
  }

  const { home, away, h2h } = data

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4 border border-ink bg-paper p-6">
        <div>
          <p className="font-serif text-xl font-bold text-navy">{home.name}</p>
          <p className="font-mono text-xs tabular-nums text-muted-brown">Elo {Math.round(home.elo)}</p>
        </div>
        <p className="font-mono text-xs uppercase tracking-widest text-muted-brown">vs</p>
        <div className="text-right">
          <p className="font-serif text-xl font-bold text-brick">{away.name}</p>
          <p className="font-mono text-xs tabular-nums text-muted-brown">Elo {Math.round(away.elo)}</p>
        </div>
      </div>

      <div className="border border-ink bg-paper p-6">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-serif text-lg font-bold">Head to Head · {h2h.played} played</h3>
          <span className="font-mono text-xs tabular-nums text-muted-brown">
            Avg goals {h2h.avgGoals.toFixed(2)}
          </span>
        </div>
        <RecordBar
          homeWins={h2h.homeWins}
          draws={h2h.draws}
          awayWins={h2h.awayWins}
        />

        {h2h.matches.length > 0 && (
          <table className="mt-6 w-full font-mono text-sm">
            <thead>
              <tr className="text-xs uppercase tracking-wider text-muted-brown">
                <th className="py-2 text-left font-medium">Date</th>
                <th className="py-2 text-left font-medium">Season</th>
                <th className="py-2 text-left font-medium">Match</th>
                <th className="py-2 text-right font-medium">Score</th>
              </tr>
            </thead>
            <tbody>
              {h2h.matches.map((m, i) => (
                <tr key={i} className="border-t border-rule/60">
                  <td className="py-1.5 tabular-nums text-muted-brown">{m.date}</td>
                  <td className="py-1.5 text-muted-brown">{m.season}</td>
                  <td className="py-1.5 font-bold">
                    {m.home.name} vs {m.away.name}
                  </td>
                  <td className="py-1.5 text-right font-bold tabular-nums">
                    {m.homeGoals} - {m.awayGoals}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="grid gap-6 sm:grid-cols-2">
        <div className="border border-ink bg-paper p-6">
          <FormGuide team={home} />
        </div>
        <div className="border border-ink bg-paper p-6">
          <FormGuide team={away} />
        </div>
      </div>
    </div>
  )
}
