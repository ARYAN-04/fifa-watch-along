import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

const MOCK_EVENTS = [
  { id: 1, minute: 12, event_type: 'GOAL', team: 'Argentina', team_id: 1, player_name: 'L. Messi', assist_name: 'A. Di María', detail: 'Left-footed shot from outside the box' },
  { id: 2, minute: 35, event_type: 'YELLOW_CARD', team: 'Canada', team_id: 2, player_name: 'A. Davies', assist_name: '', detail: '' },
  { id: 3, minute: 52, event_type: 'GOAL', team: 'Argentina', team_id: 1, player_name: 'J. Álvarez', assist_name: 'L. Messi', detail: 'Tap-in from close range' },
  { id: 4, minute: 68, event_type: 'SUBSTITUTION', team: 'Canada', team_id: 2, player_name: 'J. David ← C. Larin', assist_name: '', detail: '' },
];

export async function GET() {
  try {
    const res = await fetch(`${getBackendUrl()}/api/events/`, { next: { revalidate: 0 } });
    if (!res.ok) return NextResponse.json(MOCK_EVENTS);
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json(MOCK_EVENTS);
  }
}
