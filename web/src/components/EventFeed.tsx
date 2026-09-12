import { useQuery } from '@tanstack/react-query'
import { api, type MatchEvent } from '../lib/api'

const eventEmoji: Record<string, string> = {
  GOAL: '⚽',
  YELLOW_CARD: '🟨',
  CARD: '🟨',
  RED_CARD: '🟥',
  SUBSTITUTION: '🔄',
  SUB: '🔄',
}

const eventLabel: Record<string, string> = {
  GOAL: 'Goal',
  YELLOW_CARD: 'Yellow Card',
  CARD: 'Card',
  RED_CARD: 'Red Card',
  SUBSTITUTION: 'Substitution',
  SUB: 'Substitution',
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
    <li className="grid grid-cols-[40px_1fr] gap-3 py-3 font-mono text-xs">
      <span className="font-serif text-base font-bold text-brick">{event.minute}&apos;</span>
      <div>
        <span>{eventEmoji[event.type] ?? '•'}</span>{' '}
        <span className="font-serif text-sm font-bold">{event.player ?? 'Unknown'}</span>{' '}
        <span className="text-[10px] uppercase tracking-wider text-muted-brown">
          {eventLabel[event.type] ?? event.type} ({team})
        </span>
        {event.detail ? <div className="mt-1 text-[10px] text-muted-brown">{event.detail}</div> : null}
      </div>
    </li>
  )
}

export function EventFeed({
  matchId,
  homeName,
  awayName,
  upto = null,
}: {
  matchId: number
  homeName: string
  awayName: string
  upto?: number | null
}) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['matchEvents', matchId],
    queryFn: () => api.getMatchEvents(matchId),
    staleTime: 30_000,
  })

  if (isPending) {
    return <div className="h-32 animate-pulse border border-ink bg-paper2/30" />
  }

  if (isError || !data) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        Failed to load events.
      </div>
    )
  }

  const visible = upto == null ? data.events : data.events.filter((e) => e.minute <= upto)
  const sorted = [...visible].sort((a, b) => b.minute - a.minute)

  if (sorted.length === 0) {
    return <div className="py-8 text-center font-mono text-sm text-muted-brown">No events yet</div>
  }

  return (
    <ul className="divide-y divide-dotted divide-rule">
      {sorted.map((e, i) => (
        <EventRow key={`${e.minute}-${e.type}-${i}`} event={e} homeName={homeName} awayName={awayName} />
      ))}
    </ul>
  )
}
