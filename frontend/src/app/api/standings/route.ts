import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

const MOCK_STANDINGS = {
  'A': [
    { position: 1, team: 'Argentina', team_id: 1, played: 1, won: 1, drawn: 0, lost: 0, goals_for: 2, goals_against: 0, points: 3 },
    { position: 2, team: 'Canada', team_id: 2, played: 1, won: 0, drawn: 0, lost: 1, goals_for: 0, goals_against: 2, points: 0 },
  ]
};

export async function GET() {
  try {
    const res = await fetch(`${getBackendUrl()}/api/standings/`, { next: { revalidate: 0 } });
    if (!res.ok) return NextResponse.json(MOCK_STANDINGS);
    const data = await res.json();
    return NextResponse.json(data);
  } catch (err) {
    return NextResponse.json(MOCK_STANDINGS);
  }
}
