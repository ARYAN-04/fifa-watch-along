from django.http import JsonResponse
from django.shortcuts import get_object_or_404

from .models import MatchConfig, MatchEvent, WinProbabilitySnapshot, Standing, Player, Team


def health(request):
    return JsonResponse({'status': 'ok'})


def match_state(request):
    cfg = MatchConfig.objects.select_related(
        'current_match__home_team', 'current_match__away_team'
    ).first()
    if not cfg or not cfg.current_match:
        return JsonResponse({'error': 'No match configured'}, status=404)

    match = cfg.current_match
    home = match.home_team
    away = match.away_team

    return JsonResponse({
        'match_id': match.id,
        'status': match.status,
        'home_team': {
            'id': home.id,
            'name': home.name,
            'short_name': home.short_name,
            'flag_url': home.flag_url,
            'pre_match_elo': home.pre_match_elo,
        },
        'away_team': {
            'id': away.id,
            'name': away.name,
            'short_name': away.short_name,
            'flag_url': away.flag_url,
            'pre_match_elo': away.pre_match_elo,
        },
        'home_score': match.home_score,
        'away_score': match.away_score,
        'kickoff_utc': match.kickoff_utc.isoformat(),
        'stage': match.stage,
        'venue': match.venue,
    })


def win_probability(request):
    cfg = MatchConfig.objects.first()
    if not cfg or not cfg.current_match:
        return JsonResponse({'error': 'No match configured'}, status=404)

    snapshots = WinProbabilitySnapshot.objects.filter(
        match=cfg.current_match
    ).order_by('minute')
    current = snapshots.last()
    history = list(snapshots.values(
        'minute', 'home_win_prob', 'draw_prob', 'away_win_prob', 'score_diff', 'xg_diff_approx'
    ))

    return JsonResponse({
        'current': {
            'home_win': current.home_win_prob,
            'draw': current.draw_prob,
            'away_win': current.away_win_prob,
        } if current else None,
        'history': history,
    })


def events(request):
    cfg = MatchConfig.objects.first()
    if not cfg or not cfg.current_match:
        return JsonResponse({'error': 'No match configured'}, status=404)

    match_events = MatchEvent.objects.filter(
        match=cfg.current_match
    ).select_related('team').order_by('minute')

    return JsonResponse([{
        'id': e.id,
        'minute': e.minute,
        'event_type': e.event_type,
        'team': e.team.name if e.team else None,
        'team_id': e.team.id if e.team else None,
        'player_name': e.player_name,
        'assist_name': e.assist_name,
        'detail': e.detail,
    } for e in match_events], safe=False)


def standings(request):
    standings_qs = Standing.objects.select_related('team').order_by('group', 'position')

    groups = {}
    for s in standings_qs:
        g = s.group
        if g not in groups:
            groups[g] = []
        groups[g].append({
            'position': s.position,
            'team': s.team.name,
            'team_id': s.team.id,
            'played': s.played,
            'won': s.won,
            'drawn': s.drawn,
            'lost': s.lost,
            'goals_for': s.goals_for,
            'goals_against': s.goals_against,
            'points': s.points,
        })

    return JsonResponse(groups)


def players(request, team_id):
    team = get_object_or_404(Team, id=team_id)
    player_list = team.players.all().order_by('-overall_rating').values(
        'id', 'name', 'position', 'overall_rating',
        'pace', 'shooting', 'passing', 'dribbling',
        'defending', 'physical', 'skill_moves', 'weak_foot',
    )
    return JsonResponse({
        'team': team.name,
        'team_id': team.id,
        'players': list(player_list),
    })
