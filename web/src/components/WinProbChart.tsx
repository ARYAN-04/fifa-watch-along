import { useQuery } from '@tanstack/react-query'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { api, type WinProbabilitySnapshot } from '../lib/api'

const SERIES = [
  { key: 'home', label: 'Home', color: '#34d399' },
  { key: 'draw', label: 'Draw', color: '#a1a1aa' },
  { key: 'away', label: 'Away', color: '#f87171' },
] as const

function toChartData(
  preMatch: { home: number; draw: number; away: number } | null,
  snapshots: WinProbabilitySnapshot[],
) {
  const points = snapshots.map((s) => ({
    minute: s.minute,
    home: s.home * 100,
    draw: s.draw * 100,
    away: s.away * 100,
  }))
  if (preMatch && (points.length === 0 || points[0].minute > 0)) {
    points.unshift({ minute: 0, ...preMatch })
  }
  return points
}

function formatPct(v: number) {
  return `${Math.round(v)}%`
}

export function WinProbChart({ matchId }: { matchId: number }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['winProbability', matchId],
    queryFn: () => api.getWinProbability(matchId),
    staleTime: 30_000,
  })

  if (isPending) {
    return <div className="h-64 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900" />
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        Failed to load win probability.
      </div>
    )
  }

  const chartData = toChartData(data.preMatch, data.snapshots)

  if (chartData.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        No probability data yet.
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-4">
      <div className="mb-3 flex items-center gap-4">
        <h3 className="text-sm font-semibold uppercase tracking-widest text-zinc-400">
          Win Probability
        </h3>
        <div className="flex gap-3 text-xs text-zinc-400">
          {SERIES.map((s) => (
            <span key={s.key} className="flex items-center gap-1.5">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: s.color }} />
              {s.label}
            </span>
          ))}
        </div>
      </div>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
            <defs>
              {SERIES.map((s) => (
                <linearGradient key={s.key} id={`wp-${s.key}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={s.color} stopOpacity={0.35} />
                  <stop offset="100%" stopColor={s.color} stopOpacity={0.05} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid stroke="#27272a" strokeDasharray="3 3" />
            <XAxis
              dataKey="minute"
              stroke="#71717a"
              tick={{ fontSize: 12 }}
              tickFormatter={(m: number) => `${m}'`}
            />
            <YAxis
              domain={[0, 100]}
              stroke="#71717a"
              tick={{ fontSize: 12 }}
              tickFormatter={formatPct}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#18181b',
                border: '1px solid #3f3f46',
                borderRadius: 8,
                fontSize: 12,
                fontVariantNumeric: 'tabular-nums',
              }}
              labelFormatter={(m) => `Minute ${m}`}
              formatter={(value) => formatPct(Number(value))}
            />
            {SERIES.map((s) => (
              <Area
                key={s.key}
                type="monotone"
                dataKey={s.key}
                name={s.label}
                stroke={s.color}
                fill={`url(#wp-${s.key})`}
                strokeWidth={2}
                isAnimationActive={false}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
