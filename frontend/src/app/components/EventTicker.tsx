'use client';

interface MatchEvent {
  id: number;
  minute: number;
  event_type: string;
  team: string | null;
  player_name: string;
  assist_name?: string;
}

interface Props {
  events: MatchEvent[];
}

const eventEmoji: Record<string, string> = {
  GOAL: '⚽',
  YELLOW_CARD: '🟨',
  RED_CARD: '🟥',
  SUBSTITUTION: '🔄',
};

const eventLabel: Record<string, string> = {
  GOAL: 'Goal',
  YELLOW_CARD: 'Yellow Card',
  RED_CARD: 'Red Card',
  SUBSTITUTION: 'Substitution',
};

export default function EventTicker({ events }: Props) {
  if (events.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-muted-brown text-sm py-8 font-mono">
        No events yet
      </div>
    );
  }

  // Show newest events first (like a broadsheet print feed of logs)
  const sortedEvents = [...events].sort((a, b) => b.minute - a.minute);

  return (
    <ul className="divide-y divide-dotted divide-rule">
      {sortedEvents.map((event) => {
        const emoji = eventEmoji[event.event_type] ?? '•';
        const label = eventLabel[event.event_type] ?? event.event_type;
        return (
          <li key={event.id} className="grid grid-cols-[40px_1fr] gap-3 py-3 text-xs text-ink font-mono align-top">
            <div className="font-serif font-bold text-base text-brick leading-none">{event.minute}&apos;</div>
            <div>
              <div className="leading-tight">
                <span className="mr-1">{emoji}</span>
                <span className="font-serif font-bold text-sm text-ink">{event.player_name}</span>
                <span className="text-muted-brown text-[10px] ml-1 uppercase tracking-wider">
                  {label} {event.team ? `(${event.team})` : ''}
                </span>
              </div>
              {event.event_type === 'GOAL' && event.assist_name && (
                <div className="text-[10px] text-muted-brown mt-1">
                  Assist — {event.assist_name}
                </div>
              )}
            </div>
          </li>
        );
      })}
    </ul>
  );
}
