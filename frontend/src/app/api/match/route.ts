import { NextRequest, NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

const MOCK_MATCH = {
  match_id: 1,
  status: 'IN_PLAY',
  home_team: {
    id: 1,
    name: 'Argentina',
    short_name: 'ARG',
    flag_url: '',
    pre_match_elo: 2140,
  },
  away_team: {
    id: 2,
    name: 'Canada',
    short_name: 'CAN',
    flag_url: '',
    pre_match_elo: 1780,
  },
  home_score: 2,
  away_score: 0,
  kickoff_utc: '2026-06-10T16:00:00+00:00',
  stage: 'Group Stage',
  venue: 'MetLife Stadium',
};

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = request.nextUrl;
    const minute = searchParams.get('minute');
    let url = `${getBackendUrl()}/api/match/`;
    if (minute) url += `?minute=${minute}`;
    const res = await fetch(url, { next: { revalidate: 0 } });
    if (!res.ok) return NextResponse.json(MOCK_MATCH);
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json(MOCK_MATCH);
  }
}
