import { useQuery } from '@tanstack/react-query'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { api, type WinProbabilitySnapshot } from '../lib/api'

const SERIES = [
  { key: 'home', label: 'Home', color: 'var(--navy)' },
  { key: 'draw', label: 'Draw', color: 'var(--muted)' },
  { key: 'away', label: 'Away', color: 'var(--red)' },
] as const

function toChartData(
  preMatch: { home: number; draw: number; away: number } | null,
  snapshots: WinProbabilitySnapshot[],
  upto: number | null,
) {
  const visible = upto == null ? snapshots : snapshots.filter((s) => s.minute <= upto)
  const points = visible.map((s) => ({
    minute: s.minute,
    home: s.home * 100,
    draw: s.draw * 100,
    away: s.away * 100,
  }))
  if (preMatch && (points.length === 0 || points[0].minute > 0)) {
    points.unshift({ minute: 0, home: preMatch.home * 100, draw: preMatch.draw * 100, away: preMatch.away * 100 })
  }
  return points
}

function formatPct(v: number) {
  return `${Math.round(v)}%`
}

export function WinProbChart({
  matchId,
  homeName,
  awayName,
  upto = null,
}: {
  matchId: number
  homeName: string
  awayName: string
  upto?: number | null
}) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['winProbability', matchId],
    queryFn: () => api.getWinProbability(matchId),
    staleTime: 30_000,
  })

  if (isPending) {
    return <div className="h-64 animate-pulse border border-ink bg-paper2/30" />
  }

  if (isError || !data) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        Failed to load win probability.
      </div>
    )
  }

  const chartData = toChartData(data.preMatch, data.snapshots, upto)

  if (chartData.length === 0) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        No probability data yet.
      </div>
    )
  }

  const current = chartData[chartData.length - 1]
  const tracks = [
    { label: homeName, value: current.home, color: 'var(--navy)' },
    { label: 'Draw', value: current.draw, color: 'var(--muted)' },
    { label: awayName, value: current.away, color: 'var(--red)' },
  ]

  return (
    <div>
      <div className="mb-6 space-y-3">
        {tracks.map((t) => (
          <div key={t.label}>
            <div className="flex items-baseline justify-between">
              <span className="font-serif text-sm font-bold">{t.label}</span>
              <span className="font-serif text-xl font-black tabular-nums">
                {formatPct(t.value)}
              </span>
            </div>
            <div className="relative h-1.5 bg-rule">
              <div className="absolute inset-y-0 left-0" style={{ width: `${t.value}%`, backgroundColor: t.color }} />
            </div>
          </div>
        ))}
      </div>
      <div className="h-[160px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData} margin={{ top: 10, right: 0, bottom: -25, left: 0 }}>
            <defs>
              {SERIES.map((s) => (
                <linearGradient key={s.key} id={`wp-${s.key}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={s.color} stopOpacity={0.2} />
                  <stop offset="100%" stopColor={s.color} stopOpacity={0} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid stroke="var(--rule)" strokeDasharray="2 2" vertical={false} />
            <XAxis
              dataKey="minute"
              stroke="var(--muted)"
              tick={{ fontSize: 10, fontFamily: 'Space Mono, monospace' }}
              tickLine={false}
              axisLine={false}
              tickFormatter={(m: number) => `${m}'`}
            />
            <YAxis
              domain={[0, 100]}
              stroke="var(--muted)"
              tick={{ fontSize: 10, fontFamily: 'Space Mono, monospace' }}
              tickLine={false}
              axisLine={false}
              tickFormatter={formatPct}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--paper)',
                border: '1px solid var(--ink)',
                borderRadius: 0,
                fontSize: 11,
                fontFamily: 'Space Mono, monospace',
              }}
              labelFormatter={(m) => `Minute ${m}`}
              formatter={(value) => formatPct(Number(value))}
            />
            <ReferenceLine y={50} stroke="var(--muted)" strokeDasharray="3 3" />
            {upto != null && (
              <ReferenceLine x={upto} stroke="var(--ink)" strokeDasharray="3 3" />
            )}
            <Area
              type="monotone"
              dataKey="home"
              name={homeName}
              stroke="var(--navy)"
              fill="url(#wp-home)"
              strokeWidth={2}
              isAnimationActive={false}
            />
            <Area
              type="monotone"
              dataKey="draw"
              name="Draw"
              stroke="var(--muted)"
              fill="url(#wp-draw)"
              strokeWidth={1.5}
              isAnimationActive={false}
            />
            <Area
              type="monotone"
              dataKey="away"
              name={awayName}
              stroke="var(--red)"
              fill="url(#wp-away)"
              strokeWidth={2}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
