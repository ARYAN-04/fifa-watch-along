'use client';

interface TeamInfo {
  id: number;
  name: string;
  short_name: string;
  flag_url: string;
  pre_match_elo: number;
}

interface MatchData {
  match_id: number;
  status: string;
  home_team: TeamInfo;
  away_team: TeamInfo;
  home_score: number;
  away_score: number;
  stage: string;
  venue: string;
}

interface Props {
  match: MatchData;
}

export default function MatchInfo({ match }: Props) {
  const { home_team, away_team, home_score, away_score, status } = match;

  return (
    <div className="py-2 border-b border-rule mb-6">
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-5">
        <div className="text-left">
          <div className="text-[10px] uppercase tracking-wider text-muted-brown mb-1">Home</div>
          <div className="font-serif font-bold text-2xl text-navy leading-tight">{home_team.name}</div>
          <div className="text-[10px] text-muted-brown font-mono mt-1">ELO {home_team.pre_match_elo}</div>
        </div>
        
        <div className="text-center">
          <div className="font-serif text-5xl font-black leading-none text-ink">
            {home_score}
            <span className="text-muted-brown font-light px-1.5">:</span>
            {away_score}
          </div>
          {status === 'IN_PLAY' && (
            <div className="inline-block mt-3 border border-brick text-brick text-[10px] tracking-wider py-0.5 px-2.5 uppercase font-bold transform -rotate-2">
              Live
            </div>
          )}
          {status === 'FINISHED' && (
            <div className="inline-block mt-3 border border-ink text-ink text-[10px] tracking-wider py-0.5 px-2.5 uppercase font-bold">
              FT
            </div>
          )}
          {status === 'SCHEDULED' && (
            <div className="inline-block mt-3 border border-muted-brown text-muted-brown text-[10px] tracking-wider py-0.5 px-2.5 uppercase font-bold">
              VS
            </div>
          )}
        </div>

        <div className="text-right">
          <div className="text-[10px] uppercase tracking-wider text-muted-brown mb-1">Away</div>
          <div className="font-serif font-bold text-2xl text-brick leading-tight">{away_team.name}</div>
          <div className="text-[10px] text-muted-brown font-mono mt-1">ELO {away_team.pre_match_elo}</div>
        </div>
      </div>
    </div>
  );
}
