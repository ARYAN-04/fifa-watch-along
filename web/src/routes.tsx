import { useState } from 'react'
import { Outlet, createRootRoute, createRoute, createRouter, Link, useParams } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ScoreStrip } from './components/ScoreStrip'
import { StandingsTable } from './components/StandingsTable'
import { WinProbChart } from './components/WinProbChart'
import { EventFeed } from './components/EventFeed'
import { H2HComparison } from './components/H2HComparison'
import {
  api,
  type Fixture,
  type MatchDetail,
} from './lib/api'
import type { ReactNode } from 'react'

const LEAGUES = [
  { id: 'pl', label: 'Premier League', enabled: true },
  { id: 'ucl', label: 'UCL', enabled: false },
  { id: 'laliga', label: 'La Liga', enabled: false },
  { id: 'seriea', label: 'Serie A', enabled: false },
  { id: 'bundesliga', label: 'Bundesliga', enabled: false },
  { id: 'wc26', label: 'WC 26', enabled: false },
]

const ACTIVE_LEAGUE = 'pl'

const SEASONS = ['2025-26', '2024-25', '2023-24', '2022-23', '2021-22']

function RootLayout() {
  return (
    <div className="min-h-screen">
      <header className="border-b border-zinc-800 bg-zinc-900/60 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
          <Link to="/" className="text-lg font-bold tracking-tight text-white">
            Football Hub
          </Link>
          <nav className="flex items-center gap-1">
            <Link
              to="/fixtures"
              className="rounded-md px-3 py-1.5 text-sm font-medium text-zinc-400 hover:bg-zinc-800 hover:text-white"
              activeProps={{ className: 'bg-zinc-800 text-white' }}
            >
              Fixtures
            </Link>
            <Link
              to="/compare"
              className="rounded-md px-3 py-1.5 text-sm font-medium text-zinc-400 hover:bg-zinc-800 hover:text-white"
              activeProps={{ className: 'bg-zinc-800 text-white' }}
            >
              Compare
            </Link>
            {LEAGUES.map((l) =>
              l.enabled ? (
                <Link
                  key={l.id}
                  to="/"
                  className="rounded-md px-3 py-1.5 text-sm font-medium text-white bg-zinc-800"
                  activeOptions={{ exact: true }}
                >
                  {l.label}
                </Link>
              ) : (
                <span
                  key={l.id}
                  aria-disabled
                  className="cursor-not-allowed rounded-md px-3 py-1.5 text-sm font-medium text-zinc-600"
                >
                  {l.label}
                </span>
              ),
            )}
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}

const rootRoute = createRootRoute({ component: RootLayout })

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: (): ReactNode => (
    <section className="space-y-8">
      <div>
        <h1 className="mb-4 text-sm font-semibold uppercase tracking-widest text-zinc-500">
          Live Scores
        </h1>
        <ScoreStrip />
      </div>
      <StandingsTable leagueId={ACTIVE_LEAGUE} />
    </section>
  ),
})

function StatusBadge({ status }: { status: string }) {
  if (status === 'FINISHED') {
    return (
      <span className="rounded-full bg-zinc-800 px-2 py-0.5 text-xs text-zinc-400">FT</span>
    )
  }
  if (status === 'LIVE') {
    return (
      <span className="rounded-full bg-red-950 px-2 py-0.5 text-xs font-semibold text-red-400">
        LIVE
      </span>
    )
  }
  return (
    <span className="rounded-full bg-zinc-800 px-2 py-0.5 text-xs text-zinc-400">{status}</span>
  )
}

const dateFmt = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short' })
const timeFmt = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' })

function FixtureRow({ fixture }: { fixture: Fixture }) {
  const played = fixture.status === 'FINISHED'
  const cells = (
    <>
      <td className="px-3 py-1.5 whitespace-nowrap text-zinc-500">
        {dateFmt.format(new Date(fixture.kickoff))}
        <span className="ml-2 tabular-nums text-zinc-600">
          {timeFmt.format(new Date(fixture.kickoff))}
        </span>
      </td>
      <td className="px-2 py-1.5 truncate text-right text-zinc-200">{fixture.home.name}</td>
      <td className="px-2 py-1.5 whitespace-nowrap text-center font-semibold tabular-nums text-white">
        {played ? `${fixture.homeGoals} - ${fixture.awayGoals}` : 'vs'}
      </td>
      <td className="px-2 py-1.5 truncate text-zinc-200">{fixture.away.name}</td>
      <td className="px-2 py-1.5 text-right">
        <StatusBadge status={fixture.status} />
      </td>
    </>
  )
  if (!played) {
    return <tr className="border-t border-zinc-800/60">{cells}</tr>
  }
  return (
    <tr className="cursor-pointer border-t border-zinc-800/60 hover:bg-zinc-800/40">
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
  const [season, setSeason] = useState(SEASONS[0])
  const { data, isPending, isError } = useQuery({
    queryKey: ['fixtures', ACTIVE_LEAGUE, season],
    queryFn: () => api.getFixtures(ACTIVE_LEAGUE, season),
    staleTime: 5 * 60_000,
  })

  if (isPending) {
    return <div className="h-96 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900" />
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-sm text-zinc-400">
        Failed to load fixtures.
      </div>
    )
  }

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-sm font-semibold uppercase tracking-widest text-zinc-500">
          Fixtures &amp; Results · {data.fixtures.length} matches
        </h1>
        <select
          value={season}
          onChange={(e) => setSeason(e.target.value)}
          className="rounded-md border border-zinc-700 bg-zinc-900 px-2 py-1.5 text-sm text-zinc-200 focus:border-zinc-500 focus:outline-none"
        >
          {SEASONS.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>
      <div className="overflow-x-auto rounded-lg border border-zinc-800 bg-zinc-900">
        <table className="w-full min-w-[36rem] text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wider text-zinc-500">
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
    return <div className="h-32 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900" />
  }

  if (isError || !match) {
    return (
      <div className="rounded-lg border border-dashed border-zinc-700 p-12 text-center">
        <h2 className="text-xl font-semibold text-zinc-300">Match not found</h2>
        <p className="mt-2 text-sm text-zinc-500">The requested match could not be loaded.</p>
      </div>
    )
  }

  return <MatchDetailInner matchId={matchId} match={match} />
}

function MatchDetailInner({ matchId, match }: { matchId: number; match: MatchDetail }) {
  const kickoff = new Date(match.utcKickoff).toLocaleString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
  const live = match.status === 'LIVE' || match.status === 'IN_PLAY'

  return (
    <section className="space-y-6">
      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-6">
        <div className="flex items-center justify-center gap-2">
          <StatusBadge status={live ? 'LIVE' : match.status} />
          {live && match.minute != null && (
            <span className="text-xs tabular-nums text-red-400">{match.minute}'</span>
          )}
          <span className="text-xs tabular-nums text-zinc-500">
            {match.season} · {kickoff}
          </span>
        </div>
        <div className="mt-4 flex items-center justify-between gap-6">
          <span className="flex-1 truncate text-right text-lg font-semibold text-zinc-100">
            {match.home.name}
          </span>
          <span className="font-sans text-5xl font-bold tabular-nums text-white">
            {match.homeGoals ?? '-'} - {match.awayGoals ?? '-'}
          </span>
          <span className="flex-1 truncate text-lg font-semibold text-zinc-100">
            {match.away.name}
          </span>
        </div>
      </div>

      <WinProbChart matchId={matchId} />
      <EventFeed matchId={matchId} homeName={match.home.name} awayName={match.away.name} />
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
      <span className="text-xs font-semibold uppercase tracking-widest text-zinc-500">
        {label}
      </span>
      <input
        list={`teams-${label}`}
        placeholder="Search team…"
        defaultValue={
          value != null ? (options.find((t) => t.id === value)?.name ?? '') : ''
        }
        onChange={(e) => {
          const found = options.find((t) => t.name === e.target.value)
          onChange(found?.id ?? null)
        }}
        className="w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-zinc-500 focus:outline-none"
      />
      <datalist id={`teams-${label}`}>
        {options.map((t) => (
          <option key={t.id} value={t.name} />
        ))}
      </datalist>
    </label>
  )
}

function ComparePage() {
  const [homeId, setHomeId] = useState<number | null>(null)
  const [awayId, setAwayId] = useState<number | null>(null)

  const { data } = useQuery({
    queryKey: ['standings', ACTIVE_LEAGUE],
    queryFn: () => api.getStandings(ACTIVE_LEAGUE),
    staleTime: 10 * 60_000,
  })

  const teams = data?.standings.map((r) => r.team) ?? []

  return (
    <section className="space-y-6">
      <h1 className="text-sm font-semibold uppercase tracking-widest text-zinc-500">
        Head-to-Head Comparison
      </h1>
      <div className="grid gap-4 sm:grid-cols-2">
        <TeamPicker label="Home" value={homeId} onChange={setHomeId} options={teams} />
        <TeamPicker label="Away" value={awayId} onChange={setAwayId} options={teams} />
      </div>
      {homeId == null || awayId == null ? (
        <div className="rounded-lg border border-dashed border-zinc-700 p-12 text-center text-sm text-zinc-500">
          Pick two teams to compare.
        </div>
      ) : homeId === awayId ? (
        <div className="rounded-lg border border-dashed border-zinc-700 p-12 text-center text-sm text-zinc-500">
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
  component: FixturesPage,
})

const matchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/match/$id',
  component: MatchDetailPage,
})

const routeTree = rootRoute.addChildren([indexRoute, compareRoute, fixturesRoute, matchRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
