'use client';

interface MatchEvent {
  id: number;
  minute: number;
  event_type: string;
  team: string | null;
  player_name: string;
}

interface Props {
  events: MatchEvent[];
}

const eventConfig: Record<string, { label: string; dot: string }> = {
  GOAL: { label: 'Goal', dot: 'bg-emerald-500' },
  YELLOW_CARD: { label: 'Yellow Card', dot: 'bg-yellow-400' },
  RED_CARD: { label: 'Red Card', dot: 'bg-red-600' },
  SUBSTITUTION: { label: 'Substitution', dot: 'bg-sky-500' },
};

export default function EventTicker({ events }: Props) {
  if (events.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        No events yet
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto pr-2 space-y-4 scrollbar-hide">
      {events.map((event) => {
        const cfg = eventConfig[event.event_type] ?? { label: event.event_type, dot: 'bg-zinc-500' };
        return (
          <div
            key={event.id}
            className="flex gap-4 items-start pb-4 border-b border-zinc-800/50 last:border-0"
          >
            <div className="flex items-center gap-2 min-w-0">
              <span className={`w-2 h-2 rounded-full ${cfg.dot} flex-shrink-0`} />
              <span className="text-rose-500 font-mono text-sm tabular-nums">{event.minute}&apos;</span>
            </div>
            <div className="min-w-0">
              <div className="font-medium text-zinc-100 truncate">{cfg.label}</div>
              <div className="text-sm text-zinc-400 truncate">
                {event.player_name ? `${event.player_name}${event.team ? ` (${event.team})` : ''}` : event.team ?? ''}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
