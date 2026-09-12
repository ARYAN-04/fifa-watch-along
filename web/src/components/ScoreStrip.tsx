import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { api, type LiveMatch } from '../lib/api'

function MatchCard({ match }: { match: LiveMatch }) {
  const live = match.status === 'LIVE' || match.status === 'IN_PLAY'
  return (
    <Link
      to="/match/$id"
      params={{ id: String(match.id) }}
      className="flex min-w-[15rem] shrink-0 flex-col gap-3 border border-ink bg-paper p-4 transition-colors hover:bg-paper2/50"
    >
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] uppercase tracking-widest text-muted-brown">
          {match.externalId}
        </span>
        {live ? (
          <span className="inline-block -rotate-2 border border-brick px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-brick">
            Live {match.minute != null ? `${match.minute}'` : ''}
          </span>
        ) : (
          <span className="inline-block border border-muted-brown px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-muted-brown">
            {match.status}
          </span>
        )}
      </div>
      <div className="flex items-center justify-between gap-4">
        <span className="truncate font-mono text-sm">{match.home.name}</span>
        <span className="font-serif text-2xl font-black tabular-nums">
          {match.homeGoals} - {match.awayGoals}
        </span>
        <span className="truncate text-right font-mono text-sm">{match.away.name}</span>
      </div>
    </Link>
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
          <div key={i} className="h-28 min-w-[15rem] shrink-0 animate-pulse border border-ink bg-paper2/30" />
        ))}
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        Failed to load live scores. Retrying in 15 seconds.
      </div>
    )
  }

  if (data.matches.length === 0) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
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
