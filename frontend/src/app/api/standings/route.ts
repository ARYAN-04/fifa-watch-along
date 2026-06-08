import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

export async function GET() {
  try {
    const res = await fetch(`${getBackendUrl()}/api/standings/`, { next: { revalidate: 0 } });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    return NextResponse.json(
      { error: 'Backend unavailable' },
      { status: 502 }
    );
  }
}
