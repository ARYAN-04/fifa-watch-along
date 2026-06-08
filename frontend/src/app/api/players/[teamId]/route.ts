import { NextRequest, NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ teamId: string }> }
) {
  try {
    const { teamId } = await params;
    const res = await fetch(
      `${getBackendUrl()}/api/players/${teamId}/`,
      { next: { revalidate: 0 } }
    );
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    return NextResponse.json(
      { error: 'Backend unavailable' },
      { status: 502 }
    );
  }
}
