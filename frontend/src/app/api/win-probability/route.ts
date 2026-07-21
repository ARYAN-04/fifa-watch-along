import { NextRequest, NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

const MOCK_HISTORY = [
  { minute: 0, home_win_prob: 0.50, draw_prob: 0.25, away_win_prob: 0.25, score_diff: 0, xg_diff_approx: 0.0 },
  { minute: 15, home_win_prob: 0.55, draw_prob: 0.24, away_win_prob: 0.21, score_diff: 0, xg_diff_approx: 0.2 },
  { minute: 30, home_win_prob: 0.70, draw_prob: 0.18, away_win_prob: 0.12, score_diff: 1, xg_diff_approx: 0.7 },
  { minute: 45, home_win_prob: 0.78, draw_prob: 0.14, away_win_prob: 0.08, score_diff: 1, xg_diff_approx: 0.9 },
  { minute: 60, home_win_prob: 0.90, draw_prob: 0.06, away_win_prob: 0.04, score_diff: 2, xg_diff_approx: 1.3 },
  { minute: 75, home_win_prob: 0.92, draw_prob: 0.05, away_win_prob: 0.03, score_diff: 2, xg_diff_approx: 1.4 },
];

const MOCK_WIN_PROB = {
  current: { home_win: 0.92, draw: 0.05, away_win: 0.03 },
  history: MOCK_HISTORY,
};

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = request.nextUrl;
    const minute = searchParams.get('minute');
    let url = `${getBackendUrl()}/api/win-probability/`;
    if (minute) url += `?minute=${minute}`;
    const res = await fetch(url, { next: { revalidate: 0 } });
    if (!res.ok) return NextResponse.json(MOCK_WIN_PROB);
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json(MOCK_WIN_PROB);
  }
}
