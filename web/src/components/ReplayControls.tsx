import { Pause, Play, SkipBack, SkipForward } from 'lucide-react'

export interface ReplayMatchOption {
  id: number
  home: string
  away: string
  league: string
  homeGoals: number | null
  awayGoals: number | null
}

interface ReplayControlsProps {
  currentMinute: number
  maxMinute: number
  isPlaying: boolean
  speed: number
  currentMatchId: number
  availableMatches: ReplayMatchOption[]
  onPlayPause: () => void
  onSpeedChange: (speed: number) => void
  onScrub: (minute: number) => void
  onSkipToStart: () => void
  onSkipToEnd: () => void
  onSwitchMatch: (matchId: number) => void
}

const SPEEDS = [0.5, 1, 2, 5, 10]

export function ReplayControls({
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
  const current = availableMatches.find((m) => m.id === currentMatchId)
  return (
    <div className="border border-ink bg-paper2/30 p-3 font-mono text-xs md:p-4">
      <div className="flex flex-wrap items-center gap-2 md:gap-3">
        <span className="min-w-[70px] text-sm font-bold uppercase tracking-wider text-brick">
          Replay{' '}
          <span className="font-mono font-normal tracking-normal text-ink">
            {Math.floor(currentMinute)}&apos;
          </span>
        </span>

        <button
          onClick={onSkipToStart}
          className="border border-ink p-1.5 transition-colors hover:bg-ink hover:text-paper"
          aria-label="Skip to start"
        >
          <SkipBack className="h-3.5 w-3.5" />
        </button>

        <button
          onClick={onPlayPause}
          className="border border-ink p-1.5 transition-colors hover:bg-ink hover:text-paper"
          aria-label={isPlaying ? 'Pause' : 'Play'}
        >
          {isPlaying ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
        </button>

        <button
          onClick={onSkipToEnd}
          className="border border-ink p-1.5 transition-colors hover:bg-ink hover:text-paper"
          aria-label="Skip to end"
        >
          <SkipForward className="h-3.5 w-3.5" />
        </button>

        <div className="mx-1 h-5 w-px bg-rule" />

        <div className="flex items-center gap-1">
          {SPEEDS.map((s) => (
            <button
              key={s}
              onClick={() => onSpeedChange(s)}
              className={`border px-1.5 py-1 text-[11px] transition-colors ${
                speed === s
                  ? 'border-ink bg-ink text-paper'
                  : 'border-ink/50 text-muted-brown hover:bg-ink/10'
              }`}
              aria-label={`Speed ${s}x`}
            >
              {s}x
            </button>
          ))}
        </div>

        <div className="mx-1 hidden h-5 w-px bg-rule md:block" />

        <input
          type="range"
          min={0}
          max={maxMinute}
          step={1}
          value={Math.floor(currentMinute)}
          onChange={(e) => onScrub(Number(e.target.value))}
          className="mx-2 hidden min-w-[120px] flex-1 cursor-pointer accent-ink md:block"
          aria-label="Scrub replay minute"
        />

        <div className="mx-1 h-5 w-px bg-rule" />

        <select
          value={currentMatchId}
          onChange={(e) => onSwitchMatch(Number(e.target.value))}
          className="max-w-[180px] border border-ink bg-paper px-2 py-1 text-[11px]"
          aria-label="Switch replay match"
        >
          {current == null && <option value={currentMatchId}>This match</option>}
          {availableMatches.map((m) => (
            <option key={m.id} value={m.id}>
              {m.home} {m.homeGoals ?? '-'} - {m.awayGoals ?? '-'} {m.away} · {m.league}
            </option>
          ))}
        </select>
      </div>

      <input
        type="range"
        min={0}
        max={maxMinute}
        step={1}
        value={Math.floor(currentMinute)}
        onChange={(e) => onScrub(Number(e.target.value))}
        className="mt-2 w-full cursor-pointer accent-ink md:hidden"
        aria-label="Scrub replay minute"
      />
    </div>
  )
}
