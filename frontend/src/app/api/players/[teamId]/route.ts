import { NextRequest, NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/backend';

interface PlayerEntry {
  id: number;
  name: string;
  position: string;
  overall_rating: number;
  pace: number;
  shooting: number;
  passing: number;
  dribbling: number;
  defending: number;
  physical: number;
  skill_moves: number;
  weak_foot: number;
}

const ARGENTINA_PLAYERS: PlayerEntry[] = [
  { id: 1, name: 'L. Messi', position: 'CF', overall_rating: 91, pace: 75, shooting: 92, passing: 90, dribbling: 95, defending: 34, physical: 65, skill_moves: 4, weak_foot: 4 },
  { id: 2, name: 'E. Martínez', position: 'GK', overall_rating: 88, pace: 45, shooting: 20, passing: 42, dribbling: 38, defending: 18, physical: 78, skill_moves: 1, weak_foot: 2 },
  { id: 3, name: 'C. Romero', position: 'CB', overall_rating: 87, pace: 72, shooting: 40, passing: 68, dribbling: 62, defending: 88, physical: 82, skill_moves: 2, weak_foot: 3 },
  { id: 4, name: 'J. Álvarez', position: 'ST', overall_rating: 86, pace: 80, shooting: 84, passing: 76, dribbling: 82, defending: 38, physical: 72, skill_moves: 3, weak_foot: 4 },
  { id: 5, name: 'A. Di María', position: 'RW', overall_rating: 86, pace: 78, shooting: 82, passing: 84, dribbling: 86, defending: 32, physical: 60, skill_moves: 4, weak_foot: 3 },
  { id: 6, name: 'R. De Paul', position: 'CM', overall_rating: 85, pace: 74, shooting: 72, passing: 82, dribbling: 78, defending: 68, physical: 76, skill_moves: 3, weak_foot: 3 },
  { id: 7, name: 'E. Fernández', position: 'CM', overall_rating: 84, pace: 68, shooting: 70, passing: 84, dribbling: 80, defending: 66, physical: 70, skill_moves: 3, weak_foot: 3 },
  { id: 8, name: 'N. Molina', position: 'RB', overall_rating: 84, pace: 82, shooting: 58, passing: 74, dribbling: 72, defending: 76, physical: 70, skill_moves: 3, weak_foot: 3 },
  { id: 9, name: 'L. Martínez', position: 'CB', overall_rating: 84, pace: 70, shooting: 42, passing: 66, dribbling: 58, defending: 86, physical: 84, skill_moves: 2, weak_foot: 3 },
  { id: 10, name: 'N. Otamendi', position: 'CB', overall_rating: 83, pace: 60, shooting: 52, passing: 68, dribbling: 60, defending: 84, physical: 80, skill_moves: 2, weak_foot: 3 },
  { id: 11, name: 'L. Paredes', position: 'CDM', overall_rating: 83, pace: 62, shooting: 66, passing: 78, dribbling: 74, defending: 78, physical: 76, skill_moves: 3, weak_foot: 3 },
];

const CANADA_PLAYERS: PlayerEntry[] = [
  { id: 12, name: 'A. Davies', position: 'LB', overall_rating: 83, pace: 92, shooting: 68, passing: 76, dribbling: 80, defending: 74, physical: 72, skill_moves: 4, weak_foot: 3 },
  { id: 13, name: 'J. David', position: 'ST', overall_rating: 81, pace: 82, shooting: 80, passing: 70, dribbling: 78, defending: 34, physical: 68, skill_moves: 3, weak_foot: 4 },
  { id: 14, name: 'S. Eustáquio', position: 'CM', overall_rating: 78, pace: 62, shooting: 68, passing: 76, dribbling: 72, defending: 70, physical: 74, skill_moves: 3, weak_foot: 3 },
  { id: 15, name: 'C. Larin', position: 'ST', overall_rating: 77, pace: 74, shooting: 78, passing: 64, dribbling: 70, defending: 30, physical: 76, skill_moves: 3, weak_foot: 3 },
  { id: 16, name: 'T. Buchanan', position: 'LM', overall_rating: 76, pace: 86, shooting: 66, passing: 68, dribbling: 76, defending: 54, physical: 64, skill_moves: 3, weak_foot: 3 },
  { id: 17, name: 'M. Crépeau', position: 'GK', overall_rating: 76, pace: 40, shooting: 16, passing: 38, dribbling: 34, defending: 18, physical: 70, skill_moves: 1, weak_foot: 2 },
  { id: 18, name: 'A. Johnston', position: 'RB', overall_rating: 75, pace: 74, shooting: 50, passing: 66, dribbling: 64, defending: 72, physical: 68, skill_moves: 2, weak_foot: 3 },
  { id: 19, name: 'D. Cornelius', position: 'CB', overall_rating: 74, pace: 62, shooting: 36, passing: 58, dribbling: 52, defending: 74, physical: 76, skill_moves: 2, weak_foot: 2 },
  { id: 20, name: 'M. Bombito', position: 'CB', overall_rating: 73, pace: 78, shooting: 34, passing: 56, dribbling: 54, defending: 72, physical: 78, skill_moves: 2, weak_foot: 2 },
  { id: 21, name: 'I. Koné', position: 'CM', overall_rating: 73, pace: 68, shooting: 62, passing: 70, dribbling: 68, defending: 64, physical: 66, skill_moves: 3, weak_foot: 3 },
  { id: 22, name: 'N. Sigur', position: 'CM', overall_rating: 72, pace: 66, shooting: 60, passing: 68, dribbling: 66, defending: 62, physical: 64, skill_moves: 2, weak_foot: 3 },
];

const MOCK_ROSTERS: Record<string, { team: string; team_id: number; players: PlayerEntry[] }> = {
  '1': { team: 'Argentina', team_id: 1, players: ARGENTINA_PLAYERS },
  '2': { team: 'Canada', team_id: 2, players: CANADA_PLAYERS },
};

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ teamId: string }> },
) {
  const { teamId } = await params;

  try {
    const res = await fetch(
      `${getBackendUrl()}/api/players/${teamId}/`,
      { next: { revalidate: 0 } },
    );
    if (!res.ok) {
      const fallback = MOCK_ROSTERS[teamId] ?? { team: 'Unknown', team_id: parseInt(teamId), players: [] };
      return NextResponse.json(fallback);
    }
    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    const fallback = MOCK_ROSTERS[teamId] ?? { team: 'Unknown', team_id: parseInt(teamId), players: [] };
    return NextResponse.json(fallback);
  }
}
