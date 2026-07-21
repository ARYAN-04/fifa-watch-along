'use client';

import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceLine, CartesianGrid
} from 'recharts';

interface WinProbSnapshot {
  minute: number;
  home_win_prob: number;
  draw_prob: number;
  away_win_prob: number;
}

interface Props {
  history: WinProbSnapshot[];
  homeShortName: string;
  awayShortName: string;
}

export default function WinProbChart({ history, homeShortName, awayShortName }: Props) {
  if (history.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-brown text-sm font-mono">
        No win probability data yet
      </div>
    );
  }

  const current = history[history.length - 1];
  const homePct = current ? Math.round(current.home_win_prob * 100) : 33;
  const drawPct = current ? Math.round(current.draw_prob * 100) : 33;
  const awayPct = current ? Math.round(current.away_win_prob * 100) : 34;

  const chartData = history.map((s) => ({
    time: `${s.minute}'`,
    teamA: Math.round(s.home_win_prob * 100),
    draw: Math.round(s.draw_prob * 100),
    teamB: Math.round(s.away_win_prob * 100),
  }));

  return (
    <div>
      {/* 3-way progress tracks from Broadsheet design */}
      <div className="space-y-3 mb-6">
        <div>
          <div className="flex justify-between items-baseline mb-1">
            <span className="font-serif font-bold text-sm text-navy">{homeShortName} Win</span>
            <span className="font-serif font-black text-xl text-navy">{homePct}%</span>
          </div>
          <div className="h-1.5 bg-rule relative">
            <div className="absolute top-0 left-0 h-full bg-navy" style={{ width: `${homePct}%` }} />
          </div>
        </div>

        <div>
          <div className="flex justify-between items-baseline mb-1">
            <span className="font-serif font-bold text-sm text-muted-brown">Draw</span>
            <span className="font-serif font-black text-xl text-muted-brown">{drawPct}%</span>
          </div>
          <div className="h-1.5 bg-rule relative">
            <div className="absolute top-0 left-0 h-full bg-muted-brown" style={{ width: `${drawPct}%` }} />
          </div>
        </div>

        <div>
          <div className="flex justify-between items-baseline mb-1">
            <span className="font-serif font-bold text-sm text-brick">{awayShortName} Win</span>
            <span className="font-serif font-black text-xl text-brick">{awayPct}%</span>
          </div>
          <div className="h-1.5 bg-rule relative">
            <div className="absolute top-0 left-0 h-full bg-brick" style={{ width: `${awayPct}%` }} />
          </div>
        </div>
      </div>

      {/* Chart Ledger */}
      <div className="h-[160px] w-full mt-6">
        <ResponsiveContainer width="100%" height="100%" minWidth={0}>
          <AreaChart data={chartData} margin={{ top: 10, right: 0, left: -25, bottom: 0 }}>
            <defs>
              <linearGradient id="colorHome" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--navy)" stopOpacity={0.2} />
                <stop offset="95%" stopColor="var(--navy)" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="colorDraw" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--muted)" stopOpacity={0.1} />
                <stop offset="95%" stopColor="var(--muted)" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="colorAway" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--red)" stopOpacity={0.2} />
                <stop offset="95%" stopColor="var(--red)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="var(--rule)" strokeDasharray="2 2" vertical={false} />
            <XAxis
              dataKey="time"
              stroke="var(--muted)"
              fontSize={10}
              tickLine={false}
              axisLine={false}
              fontFamily="var(--font-space-mono), monospace"
            />
            <YAxis
              stroke="var(--muted)"
              fontSize={10}
              tickLine={false}
              axisLine={false}
              tickFormatter={(val: number) => `${val}%`}
              domain={[0, 100]}
              fontFamily="var(--font-space-mono), monospace"
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--paper)',
                borderColor: 'var(--ink)',
                borderRadius: '0px',
                fontFamily: 'var(--font-space-mono), monospace',
                fontSize: '11px',
                color: 'var(--ink)'
              }}
              itemStyle={{ color: 'var(--ink)' }}
              formatter={(value, name) => {
                if (name === 'draw') return [`${value}%`, 'Draw'];
                const label = name === 'teamA' ? homeShortName : awayShortName;
                return [`${value}%`, label];
              }}
              labelFormatter={(label) => `Minute ${label}`}
            />
            <ReferenceLine y={50} stroke="var(--rule)" strokeDasharray="3 3" />
            <Area
              type="monotone"
              dataKey="teamA"
              stroke="var(--navy)"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorHome)"
              isAnimationActive={false}
            />
            <Area
              type="monotone"
              dataKey="draw"
              stroke="var(--muted)"
              strokeWidth={1.5}
              fillOpacity={1}
              fill="url(#colorDraw)"
              isAnimationActive={false}
            />
            <Area
              type="monotone"
              dataKey="teamB"
              stroke="var(--red)"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorAway)"
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
