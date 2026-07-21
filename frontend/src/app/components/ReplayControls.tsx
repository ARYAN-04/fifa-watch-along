'use client';

import { Play, Pause, SkipBack, SkipForward } from 'lucide-react';

interface MatchOption {
  match_id: number;
  home_team: string;
  away_team: string;
  stage: string;
  home_score: number;
  away_score: number;
}

interface ReplayControlsProps {
  currentMinute: number;
  maxMinute: number;
  isPlaying: boolean;
  speed: number;
  currentMatchId?: number;
  availableMatches: MatchOption[];
  onPlayPause: () => void;
  onSpeedChange: (speed: number) => void;
  onScrub: (minute: number) => void;
  onSkipToStart: () => void;
  onSkipToEnd: () => void;
  onSwitchMatch: (matchId: number) => void;
}

const SPEEDS = [0.5, 1, 2, 5, 10];

export default function ReplayControls({
  currentMinute,
  maxMinute,
  isPlaying,
  speed,
  currentMatchId,
  availableMatches,
  onPlayPause,
  onSpeedChange,
  onScrub,
  onSkipToStart,
  onSkipToEnd,
  onSwitchMatch,
}: ReplayControlsProps) {
  return (
    <div className="border border-ink bg-paper2/30 p-3 md:p-4 font-mono text-xs">
      <div className="flex flex-wrap items-center gap-2 md:gap-3">

        <span className="text-brick font-bold text-sm tracking-wider uppercase min-w-[70px]">
          REPLAY{' '}
          <span className="text-ink font-mono font-normal tracking-normal">
            {currentMinute}&apos;
          </span>
        </span>

        <button
          onClick={onSkipToStart}
          className="p-1.5 border border-ink hover:bg-ink hover:text-paper transition-colors"
          aria-label="Skip to start"
        >
          <SkipBack className="w-3.5 h-3.5" />
        </button>

        <button
          onClick={onPlayPause}
          className="p-1.5 border border-ink hover:bg-ink hover:text-paper transition-colors"
          aria-label={isPlaying ? 'Pause' : 'Play'}
        >
          {isPlaying ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}
        </button>

        <button
          onClick={onSkipToEnd}
          className="p-1.5 border border-ink hover:bg-ink hover:text-paper transition-colors"
          aria-label="Skip to end"
        >
          <SkipForward className="w-3.5 h-3.5" />
        </button>

        <div className="h-5 w-px bg-rule mx-1" />

        {SPEEDS.map((s) => (
          <button
            key={s}
            onClick={() => onSpeedChange(s)}
            className={`px-2 py-1 border text-[11px] tracking-wider font-bold transition-colors ${
              speed === s
                ? 'bg-ink text-paper border-ink'
                : 'border-ink/50 text-muted-brown hover:bg-ink/10 hover:border-ink'
            }`}
          >
            {s}x
          </button>
        ))}

        {availableMatches.length > 0 && (
          <>
            <div className="h-5 w-px bg-rule mx-1" />
            <select
              value={currentMatchId ?? ''}
              onChange={(e) => onSwitchMatch(Number(e.target.value))}
              className="bg-paper border border-ink px-2 py-1 text-[11px] font-mono text-ink
                         focus:outline-none focus:ring-1 focus:ring-ink max-w-[180px]"
            >
              {availableMatches.map((m) => (
                <option key={m.match_id} value={m.match_id}>
                  {m.home_team} {m.home_score}-{m.away_score} {m.away_team}
                </option>
              ))}
            </select>
          </>
        )}

        <div className="hidden md:block flex-1 min-w-[120px] mx-2">
          <input
            type="range"
            min={0}
            max={maxMinute}
            value={currentMinute}
            onChange={(e) => onScrub(Number(e.target.value))}
            className="w-full accent-ink cursor-pointer"
            aria-label="Match timeline"
          />
        </div>
      </div>

      <div className="mt-2 md:hidden">
        <input
          type="range"
          min={0}
          max={maxMinute}
          value={currentMinute}
          onChange={(e) => onScrub(Number(e.target.value))}
          className="w-full accent-ink cursor-pointer"
          aria-label="Match timeline"
        />
      </div>
    </div>
  );
}
