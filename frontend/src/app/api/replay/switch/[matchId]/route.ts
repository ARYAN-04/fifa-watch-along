import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

export async function POST(
  _request: Request,
  { params }: { params: Promise<{ matchId: string }> },
) {
  const { matchId } = await params;
  try {
    const backendUrl = `${getBackendUrl()}/api/replay/switch/${matchId}/`;
    const res = await fetch(backendUrl, { method: "POST", next: { revalidate: 0 } });
    if (!res.ok) return NextResponse.json({ error: "Switch failed" }, { status: 500 });
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json({ error: "Switch failed" }, { status: 500 });
  }
}
