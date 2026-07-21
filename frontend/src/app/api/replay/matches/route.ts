import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

export async function GET() {
  try {
    const res = await fetch(`${getBackendUrl()}/api/replay/matches/`, { next: { revalidate: 0 } });
    if (!res.ok) return NextResponse.json([], { status: 200 });
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json([], { status: 200 });
  }
}
