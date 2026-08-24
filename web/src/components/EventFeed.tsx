import { useQuery } from '@tanstack/react-query'
import { ArrowRightLeft, Goal, Square } from 'lucide-react'
import { api, type MatchEvent } from '../lib/api'

function EventIcon({ event }: { event: MatchEvent }) {
  switch (event.type) {
    case 'GOAL':
      return <Goal className="h-4 w-4 text-emerald-400" />
    case 'CARD':
      return <Square className="h-3.5 w-3.5 fill-yellow-400 text-yellow-400" />
    case 'SUB':
      return <ArrowRightLeft className="h-4 w-4 text-sky-400" />
    default:
      return <span className="text-xs text-zinc-500">{event.type}</span>
  }
}

function EventRow({
  event,
  homeName,
  awayName,
}: {
  event: MatchEvent
  homeName: string
  awayName: string
}) {
  const isHome = event.side.toUpperCase() === 'HOME'
  const team = isHome ? homeName : awayName
  return (
    <li
      className={`flex items-center gap-3 border-t border-zinc-800/60 px-4 py-2 ${
        isHome ? '' : 'flex-row-reverse text-right'
      }`}
    >
      <span className="w-8 shrink-0 tabular-nums text-xs text-zinc-500">{event.minute}'</span>
      <EventIcon event={event} />
      <div className={isHome ? '' : 'flex-1'}>
        <span className="text-sm font-medium text-zinc-200">{event.player}</span>{' '}
        <span className="text-xs text-zinc-500">
          {team}
          {event.detail ? ` · ${event.detail}` : ''}
        </span>
      </div>
      {isHome && <div className="flex-1" />}
    </li>
  )
}

export function EventFeed({
  matchId,
  homeName,
  awayName,
}: {
  matchId: number
  homeName: string
  awayName: string
}) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['matchEvents', matchId],
    queryFn: () => api.getMatchEvents(matchId),
    staleTime: 30_000,
  })

  if (isPending) {
    return <div className="h-32 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900" />
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        Failed to load events.
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900">
      <h3 className="border-b border-zinc-800 px-4 py-3 text-sm font-semibold uppercase tracking-widest text-zinc-400">
        Events
      </h3>
      {data.events.length === 0 ? (
        <p className="px-4 py-6 text-sm text-zinc-400">No events recorded.</p>
      ) : (
        <ul>
          {data.events.map((e, i) => (
            <EventRow key={`${e.minute}-${e.type}-${i}`} event={e} homeName={homeName} awayName={awayName} />
          ))}
        </ul>
      )}
    </div>
  )
}
