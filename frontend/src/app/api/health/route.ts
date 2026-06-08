import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

export async function GET() {
  try {
    const res = await fetch(`${getBackendUrl()}/health/`);
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json(
      { error: 'Backend unavailable' },
      { status: 502 }
    );
  }
}
