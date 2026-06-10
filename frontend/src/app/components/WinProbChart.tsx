'use client';

import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceLine,
} from 'recharts';

interface WinProbSnapshot {
  minute: number;
  home_win_prob: number;
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
      <div className="flex items-center justify-center h-[300px] text-zinc-500 text-sm">
        No win probability data yet
      </div>
    );
  }

  const chartData = history.map((s) => ({
    time: `${s.minute}'`,
    teamA: Math.round(s.home_win_prob * 100),
    teamB: Math.round(s.away_win_prob * 100),
  }));

  return (
    <div className="h-[300px] w-full">
      <ResponsiveContainer width="100%" height="100%" minWidth={0}>
        <AreaChart data={chartData} margin={{ top: 10, right: 0, left: -20, bottom: 0 }}>
          <defs>
            <linearGradient id="colorHome" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#f43f5e" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#f43f5e" stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="time"
            stroke="#52525b"
            fontSize={12}
            tickLine={false}
            axisLine={false}
          />
          <YAxis
            stroke="#52525b"
            fontSize={12}
            tickLine={false}
            axisLine={false}
            tickFormatter={(val: number) => `${val}%`}
            domain={[0, 100]}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: '#18181b',
              borderColor: '#27272a',
              borderRadius: '8px',
            }}
            itemStyle={{ color: '#e4e4e7' }}
            formatter={(value, name) => {
              const label = name === 'teamA' ? homeShortName : awayShortName;
              return [`${value}%`, label];
            }}
            labelFormatter={(label) => `Minute ${label}`}
          />
          <ReferenceLine y={50} stroke="#3f3f46" strokeDasharray="3 3" />
          <Area
            type="monotone"
            dataKey="teamA"
            stroke="#f43f5e"
            strokeWidth={3}
            fillOpacity={1}
            fill="url(#colorHome)"
            isAnimationActive={false}
          />
          <Area
            type="monotone"
            dataKey="teamB"
            stroke="#52525b"
            strokeWidth={3}
            fillOpacity={0}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
