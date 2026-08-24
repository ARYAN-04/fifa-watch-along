import { useQuery } from '@tanstack/react-query'
import { Radio } from 'lucide-react'
import { api, type LiveMatch } from '../lib/api'

function MatchCard({ match }: { match: LiveMatch }) {
  const live = match.status === 'LIVE' || match.status === 'IN_PLAY'
  return (
    <div className="flex min-w-[15rem] shrink-0 flex-col gap-3 rounded-lg border border-zinc-800 bg-zinc-900 p-4">
      <div className="flex items-center justify-between">
        <span className="text-xs text-zinc-500">{match.externalId}</span>
        {live ? (
          <span className="flex items-center gap-1.5 rounded-full bg-red-950 px-2 py-0.5 text-xs font-semibold text-red-400">
            <Radio className="h-3 w-3 animate-pulse" />
            {match.minute != null ? `${match.minute}'` : 'LIVE'}
          </span>
        ) : (
          <span className="rounded-full bg-zinc-800 px-2 py-0.5 text-xs text-zinc-400">
            {match.status}
          </span>
        )}
      </div>
      <div className="flex items-center justify-between gap-4">
        <span className="truncate text-sm text-zinc-300">{match.home.name}</span>
        <span className="font-sans text-2xl font-bold tabular-nums text-white">
          {match.homeGoals} - {match.awayGoals}
        </span>
        <span className="truncate text-right text-sm text-zinc-300">{match.away.name}</span>
      </div>
    </div>
  )
}

export function ScoreStrip() {
  const { data, isPending, isError } = useQuery({
    queryKey: ['liveScores'],
    queryFn: api.getLiveScores,
    refetchInterval: 15_000,
  })

  if (isPending) {
    return (
      <div className="flex gap-4 overflow-x-auto">
        {[0, 1, 2].map((i) => (
          <div
            key={i}
            className="h-28 min-w-[15rem] shrink-0 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900"
          />
        ))}
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        Failed to load live scores. Retrying in 15 seconds.
      </div>
    )
  }

  if (data.matches.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        No live matches right now.
      </div>
    )
  }

  return (
    <div className="flex gap-4 overflow-x-auto pb-2">
      {data.matches.map((m) => (
        <MatchCard key={m.id} match={m} />
      ))}
    </div>
  )
}
