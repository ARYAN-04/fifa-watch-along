import { useEffect, useState } from 'react'
import { Navigate, Outlet, createRootRoute, createRoute, createRouter, Link, useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Clock } from 'lucide-react'
import { ScoreStrip } from './components/ScoreStrip'
import { StandingsTable } from './components/StandingsTable'
import { WinProbChart } from './components/WinProbChart'
import { EventFeed } from './components/EventFeed'
import { H2HComparison } from './components/H2HComparison'
import { ReplayControls } from './components/ReplayControls'
import {
  api,
  type Fixture,
  type MatchDetail,
  type MatchEvent,
} from './lib/api'
import type { ReactNode } from 'react'

const LEAGUES = [
  { id: 'pl', label: 'Premier League', enabled: true },
  { id: 'ucl', label: 'UCL', enabled: false },
  { id: 'wc2026', label: 'FIFA WC 26', enabled: true },
]

const ACTIVE_LEAGUE = 'pl'

const SEASONS = ['2025-26', '2024-25', '2023-24', '2022-23', '2021-22']

function RootLayout() {
  const params = useParams({ strict: false }) as { leagueId?: string }
  const activeLeague = params.leagueId ?? 'pl'
  const tabClass = (id: string) =>
    `px-3 py-1.5 text-xs font-bold uppercase tracking-widest transition-colors ${
      activeLeague === id ? 'bg-ink text-paper' : 'text-muted-brown hover:bg-ink hover:text-paper'
    }`
  return (
    <div className="min-h-screen bg-paper font-mono text-ink">
      <header className="border-b-[3px] border-double border-ink px-4 py-6 text-center md:px-10">
        <Link to="/" className="font-serif text-4xl font-black uppercase tracking-tight md:text-5xl">
          Casual Watcher
        </Link>
        <nav className="mt-4 flex flex-wrap items-center justify-center gap-1">
          <Link
            to="/fixtures"
            search={{ league: activeLeague }}
            className="px-3 py-1.5 text-xs font-bold uppercase tracking-widest text-muted-brown hover:bg-ink hover:text-paper"
          >
            Fixtures
          </Link>
          <Link
            to="/compare"
            className="px-3 py-1.5 text-xs font-bold uppercase tracking-widest text-muted-brown hover:bg-ink hover:text-paper"
          >
            Compare
          </Link>
          {LEAGUES.map((l) =>
            l.enabled ? (
              <Link
                key={l.id}
                to="/l/$leagueId"
                params={{ leagueId: l.id }}
                className={tabClass(l.id)}
              >
                {l.label}
              </Link>
            ) : (
              <span
                key={l.id}
                aria-disabled
                title="No data yet"
                className="cursor-not-allowed px-3 py-1.5 text-xs font-bold uppercase tracking-widest text-rule"
              >
                {l.label}
              </span>
            ),
          )}
        </nav>
      </header>
      <main className="mx-auto my-8 w-[calc(100%-2rem)] max-w-[1400px] border border-ink bg-paper p-4 shadow-sm md:p-8 xl:max-w-[1600px]">
        <Outlet />
      </main>
    </div>
  )
}

const rootRoute = createRootRoute({ component: RootLayout })

function SectionTitle({ children }: { children: ReactNode }) {
  return (
    <>
      <h2 className="font-serif text-lg font-bold">{children}</h2>
      <hr className="my-2 border-t-2 border-ink" />
    </>
  )
}

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: (): ReactNode => <Navigate to="/l/$leagueId" params={{ leagueId: ACTIVE_LEAGUE }} />,
})

function LeaguePage({ leagueId }: { leagueId: string }) {
  const league = LEAGUES.find((l) => l.id === leagueId)
  if (!league) {
    return (
      <div className="border border-ink bg-paper2/30 p-12 text-center">
        <h2 className="font-serif text-xl font-bold">Unknown league</h2>
        <p className="mt-2 font-mono text-sm text-muted-brown">Pick a league above.</p>
      </div>
    )
  }
  return (
    <section className="space-y-8">
      <div>
        <div className="flex items-baseline justify-between gap-4">
          <SectionTitle>{league.label}</SectionTitle>
          <Link
            to="/fixtures"
            search={{ league: league.id }}
            className="shrink-0 font-mono text-xs font-bold uppercase tracking-widest text-brick hover:underline"
          >
            Full fixtures →
          </Link>
        </div>
        <div className="mt-4">
          <ScoreStrip />
        </div>
      </div>
      <div>
        <SectionTitle>League Table</SectionTitle>
        <div className="mt-4">
          <StandingsTable leagueId={league.id} />
        </div>
      </div>
      <ReplayArchive league={league.id} />
    </section>
  )
}

const leagueRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/l/$leagueId',
  component: function LeagueRouteComponent() {
    const { leagueId } = useParams({ from: '/l/$leagueId' })
    return <LeaguePage leagueId={leagueId} />
  },
})

function ReplayArchive({ league }: { league?: string }) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['replayMatches'],
    queryFn: api.getReplayMatches,
    staleTime: 5 * 60_000,
  })

  if (isPending || isError || !data) {
    return null
  }

  const matches =
    league == null ? data.matches : data.matches.filter((m) => m.league.toUpperCase() === league.toUpperCase())
  if (matches.length === 0) {
    return null
  }

  return (
    <div>
      <SectionTitle>Replay Archive</SectionTitle>
      <div className="mt-4 overflow-x-auto border border-ink bg-paper">
        <table className="w-full min-w-[36rem] font-mono text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wider text-muted-brown">
              <th className="px-3 py-2 text-left font-medium">Date</th>
              <th className="px-2 py-2 text-left font-medium">Competition</th>
              <th className="px-2 py-2 text-right font-medium">Home</th>
              <th className="px-2 py-2 text-center font-medium">Score</th>
              <th className="px-2 py-2 text-left font-medium">Away</th>
              <th className="px-2 py-2 text-right font-medium">Frames</th>
            </tr>
          </thead>
          <tbody>
            {matches.map((m) => (
              <tr key={m.id} className="cursor-pointer border-t border-rule/60 hover:bg-paper2/50">
                <Link
                  to="/match/$id"
                  params={{ id: String(m.id) }}
                  className="contents"
                  aria-label={`${m.home.name} vs ${m.away.name}`}
                >
                  <td className="whitespace-nowrap px-3 py-1.5 text-muted-brown">
                    {dateFmt.format(new Date(m.kickoff))}
                  </td>
                  <td className="px-2 py-1.5 text-[11px] uppercase tracking-widest text-brick">
                    {m.league}
                  </td>
                  <td className="truncate px-2 py-1.5 text-right font-bold">{m.home.name}</td>
                  <td className="whitespace-nowrap px-2 py-1.5 text-center font-serif text-lg font-black tabular-nums">
                    {m.homeGoals ?? '-'} - {m.awayGoals ?? '-'}
                  </td>
                  <td className="truncate px-2 py-1.5 font-bold">{m.away.name}</td>
                  <td className="px-2 py-1.5 text-right tabular-nums text-muted-brown">
                    {m.snapshots}
                  </td>
                </Link>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  if (status === 'FINISHED' || status === 'FT') {
    return (
      <span className="inline-block border border-ink px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider">
        FT
      </span>
    )
  }
  if (status === 'LIVE' || status === 'IN_PLAY' || status === 'Live') {
    return (
      <span className="inline-block -rotate-2 border border-brick px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-brick">
        Live
      </span>
    )
  }
  return (
    <span className="inline-block border border-muted-brown px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-muted-brown">
      {status}
    </span>
  )
}

const dateFmt = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short' })
const timeFmt = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' })

function FixtureRow({ fixture }: { fixture: Fixture }) {
  const played = fixture.status === 'FINISHED'
  const cells = (
    <>
      <td className="whitespace-nowrap px-3 py-1.5 text-muted-brown">
        {dateFmt.format(new Date(fixture.kickoff))}
        <span className="ml-2 tabular-nums text-muted-brown/70">
          {timeFmt.format(new Date(fixture.kickoff))}
        </span>
      </td>
      <td className="truncate px-2 py-1.5 text-right font-bold">{fixture.home.name}</td>
      <td className="whitespace-nowrap px-2 py-1.5 text-center font-serif text-lg font-black tabular-nums">
        {played ? `${fixture.homeGoals} - ${fixture.awayGoals}` : 'vs'}
      </td>
      <td className="truncate px-2 py-1.5 font-bold">{fixture.away.name}</td>
      <td className="px-2 py-1.5 text-right">
        <StatusBadge status={fixture.status} />
      </td>
    </>
  )
  if (!played) {
    return <tr className="border-t border-rule/60">{cells}</tr>
  }
  return (
    <tr className="cursor-pointer border-t border-rule/60 hover:bg-paper2/50">
      <Link
        to="/match/$id"
        params={{ id: String(fixture.id) }}
        className="contents"
        aria-label={`${fixture.home.name} vs ${fixture.away.name}`}
      >
        {cells}
      </Link>
    </tr>
  )
}

function FixturesPage() {
  const { league: leagueParam } = useSearch({ from: '/fixtures' })
  const [season, setSeason] = useState(SEASONS[0])
  const [league, setLeague] = useState(
    LEAGUES.some((l) => l.id === leagueParam) ? leagueParam : ACTIVE_LEAGUE,
  )
  useEffect(() => {
    if (LEAGUES.some((l) => l.id === leagueParam)) setLeague(leagueParam)
  }, [leagueParam])
  const { data, isPending, isError } = useQuery({
    queryKey: ['fixtures', league, season],
    queryFn: () => api.getFixtures(league, season),
    staleTime: 5 * 60_000,
  })

  if (isPending) {
    return <div className="h-96 animate-pulse border border-ink bg-paper2/30" />
  }

  if (isError || !data) {
    return (
      <div className="border border-ink bg-paper2/30 p-6 font-mono text-sm text-muted-brown">
        Failed to load fixtures.
      </div>
    )
  }

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-serif text-lg font-bold">
          Fixtures &amp; Results · {data.fixtures.length} matches
        </h1>
        <div className="flex gap-2">
          <select
            value={league}
            onChange={(e) => setLeague(e.target.value)}
            className="border border-ink bg-paper px-2 py-1.5 font-mono text-sm focus:outline-none"
            aria-label="Pick a league"
          >
            {LEAGUES.map((l) => (
              <option key={l.id} value={l.id}>
                {l.label}
              </option>
            ))}
          </select>
          <select
            value={season}
            onChange={(e) => setSeason(e.target.value)}
            className="border border-ink bg-paper px-2 py-1.5 font-mono text-sm focus:outline-none"
            aria-label="Pick a season"
          >
            {SEASONS.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="overflow-x-auto border border-ink bg-paper">
        <table className="w-full min-w-[36rem] font-mono text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wider text-muted-brown">
              <th className="px-3 py-2 text-left font-medium">Date</th>
              <th className="px-2 py-2 text-right font-medium">Home</th>
              <th className="px-2 py-2 text-center font-medium">Score</th>
              <th className="px-2 py-2 text-left font-medium">Away</th>
              <th className="px-2 py-2 text-right font-medium">Status</th>
            </tr>
          </thead>
          <tbody>{data.fixtures.map((f) => <FixtureRow key={f.id} fixture={f} />)}</tbody>
        </table>
      </div>
    </section>
  )
}

function scoreAtMinute(events: MatchEvent[], upto: number): [number, number] {
  let home = 0
  let away = 0
  for (const e of events) {
    if (e.minute > upto || e.type.toUpperCase() !== 'GOAL') continue
    if (e.side.toUpperCase() === 'HOME') home += 1
    else away += 1
  }
  return [home, away]
}

function MatchDetailPage() {
  const { id } = useParams({ from: '/match/$id' })
  const matchId = Number(id)
  const { data: match, isPending, isError } = useQuery({
    queryKey: ['match', matchId],
    queryFn: () => api.getMatch(matchId),
    staleTime: 30_000,
    enabled: Number.isFinite(matchId),
  })

  if (isPending) {
    return <div className="h-32 animate-pulse border border-ink bg-paper2/30" />
  }

  if (isError || !match) {
    return (
      <div className="border border-ink bg-paper2/30 p-12 text-center">
        <h2 className="font-serif text-xl font-bold">Match not found</h2>
        <p className="mt-2 font-mono text-sm text-muted-brown">The requested match could not be loaded.</p>
      </div>
    )
  }

  return <MatchDetailInner key={matchId} matchId={matchId} match={match} />
}

function MatchDetailInner({ matchId, match }: { matchId: number; match: MatchDetail }) {
  const navigate = useNavigate()
  const { data: winProb } = useQuery({
    queryKey: ['winProbability', matchId],
    queryFn: () => api.getWinProbability(matchId),
    staleTime: 30_000,
  })
  const { data: eventsData } = useQuery({
    queryKey: ['matchEvents', matchId],
    queryFn: () => api.getMatchEvents(matchId),
    staleTime: 30_000,
  })
  const { data: replayList } = useQuery({
    queryKey: ['replayMatches'],
    queryFn: api.getReplayMatches,
    staleTime: 5 * 60_000,
  })

  const snapshots = winProb?.snapshots ?? []
  const events = eventsData?.events ?? []
  const replayable = snapshots.length > 0
  const maxMinute = Math.max(
    95,
    ...snapshots.map((s) => s.minute),
    ...events.map((e) => e.minute),
  )

  const [replayMinute, setReplayMinute] = useState<number | null>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [speed, setSpeed] = useState(1)

  useEffect(() => {
    if (replayable && replayMinute === null) {
      setReplayMinute(0)
      setIsPlaying(true)
    }
  }, [replayable, replayMinute])

  useEffect(() => {
    if (!isPlaying || replayMinute === null) return
    const tick = setInterval(() => {
      setReplayMinute((m) => {
        if (m === null) return m
        const next = m + speed
        if (next >= maxMinute) {
          setIsPlaying(false)
          return maxMinute
        }
        return next
      })
    }, 1000)
    return () => clearInterval(tick)
  }, [isPlaying, speed, maxMinute, replayMinute === null])

  const replaying = replayMinute !== null
  const shownMinute = replaying ? Math.floor(replayMinute) : null
  const [foldHome, foldAway] = replaying ? scoreAtMinute(events, replayMinute) : [null, null]
  const hasGoals = events.some((e) => e.type.toUpperCase() === 'GOAL')
  const homeGoals = replaying && hasGoals ? foldHome : match.homeGoals
  const awayGoals = replaying && hasGoals ? foldAway : match.awayGoals
  const status = replaying ? (replayMinute >= maxMinute ? 'FINISHED' : 'IN_PLAY') : match.status
  const minute = replaying ? shownMinute : match.minute

  const kickoff = new Date(match.utcKickoff).toLocaleString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })

  return (
    <section className="space-y-6">
      {replayable && replayMinute !== null && (
        <ReplayControls
          currentMinute={replayMinute}
          maxMinute={maxMinute}
          isPlaying={isPlaying}
          speed={speed}
          currentMatchId={matchId}
          availableMatches={(replayList?.matches ?? []).map((m) => ({
            id: m.id,
            home: m.home.name,
            away: m.away.name,
            league: m.league,
            homeGoals: m.homeGoals,
            awayGoals: m.awayGoals,
          }))}
          onPlayPause={() => setIsPlaying((p) => !p)}
          onSpeedChange={setSpeed}
          onScrub={(m) => {
            setReplayMinute(m)
            setIsPlaying(false)
          }}
          onSkipToStart={() => {
            setReplayMinute(0)
            setIsPlaying(false)
          }}
          onSkipToEnd={() => {
            setReplayMinute(maxMinute)
            setIsPlaying(false)
          }}
          onSwitchMatch={(id) => navigate({ to: '/match/$id', params: { id: String(id) } })}
        />
      )}

      <div className="border-b border-rule py-2">
        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-5">
          <div className="text-right">
            <p className="text-[10px] uppercase tracking-wider text-muted-brown">Home</p>
            <p className="font-serif text-2xl font-bold text-navy">{match.home.name}</p>
          </div>
          <div className="text-center">
            <p className="font-serif text-5xl font-black tabular-nums">
              {homeGoals ?? '-'}<span className="px-1.5 font-light text-muted-brown">:</span>{awayGoals ?? '-'}
            </p>
            <StatusBadge status={status} />
            <p className="mt-2 font-mono text-[10px] tabular-nums text-muted-brown">
              {minute != null ? `${minute}' · ` : ''}{match.season} · {kickoff}
            </p>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-wider text-muted-brown">Away</p>
            <p className="font-serif text-2xl font-bold text-brick">{match.away.name}</p>
          </div>
        </div>
      </div>

      <div className="mt-8">
        <SectionTitle>Win Probability</SectionTitle>
        <div className="mt-4">
          <WinProbChart matchId={matchId} homeName={match.home.name} awayName={match.away.name} upto={replayMinute} />
        </div>
      </div>

      <div className="mt-10">
        <SectionTitle>Match Log</SectionTitle>
        <div className="mt-2">
          <EventFeed matchId={matchId} homeName={match.home.name} awayName={match.away.name} upto={replayMinute} />
        </div>
      </div>

      <footer className="flex items-center justify-between border-t border-ink py-4 font-mono text-[9px] font-bold uppercase tracking-widest text-muted-brown">
        <span>Data: football-data.org / openfootball</span>
        <span className="flex items-center gap-1.5">
          <Clock className="h-2.5 w-2.5" />
          {replaying ? `Replay ${shownMinute}' · ${speed}x` : status}
        </span>
      </footer>
    </section>
  )
}

function TeamPicker({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: number | null
  onChange: (id: number | null) => void
  options: { id: number; name: string }[]
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="font-mono text-xs font-bold uppercase tracking-widest text-muted-brown">
        {label}
      </span>
      <select
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))}
        className="w-full border border-ink bg-paper px-3 py-2 font-mono text-sm focus:outline-none"
      >
        <option value="">Pick a team…</option>
        {options.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name}
          </option>
        ))}
      </select>
    </label>
  )
}

function ComparePage() {
  const [league, setLeague] = useState(ACTIVE_LEAGUE)
  const [homeId, setHomeId] = useState<number | null>(null)
  const [awayId, setAwayId] = useState<number | null>(null)

  const { data } = useQuery({
    queryKey: ['standings', league],
    queryFn: () => api.getStandings(league),
    staleTime: 10 * 60_000,
  })

  const teams = data?.standings.map((r) => r.team) ?? []

  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <SectionTitle>Head-to-Head Comparison</SectionTitle>
        <select
          value={league}
          onChange={(e) => {
            setLeague(e.target.value)
            setHomeId(null)
            setAwayId(null)
          }}
          className="border border-ink bg-paper px-2 py-1.5 font-mono text-sm focus:outline-none"
          aria-label="Pick a league"
        >
          {LEAGUES.map((l) => (
            <option key={l.id} value={l.id}>
              {l.label}
            </option>
          ))}
        </select>
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <TeamPicker label="Home" value={homeId} onChange={setHomeId} options={teams} />
        <TeamPicker label="Away" value={awayId} onChange={setAwayId} options={teams} />
      </div>
      {homeId == null || awayId == null ? (
        <div className="border border-dashed border-ink p-12 text-center font-mono text-sm text-muted-brown">
          Pick two teams to compare.
        </div>
      ) : homeId === awayId ? (
        <div className="border border-dashed border-ink p-12 text-center font-mono text-sm text-muted-brown">
          Pick two different teams.
        </div>
      ) : (
        <H2HComparison homeId={homeId} awayId={awayId} />
      )}
    </section>
  )
}

const compareRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/compare',
  component: ComparePage,
})

const fixturesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/fixtures',
  validateSearch: (search: Record<string, unknown>) => ({
    league: typeof search.league === 'string' ? search.league : ACTIVE_LEAGUE,
  }),
  component: FixturesPage,
})

const matchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/match/$id',
  component: MatchDetailPage,
})

const routeTree = rootRoute.addChildren([indexRoute, leagueRoute, compareRoute, fixturesRoute, matchRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
