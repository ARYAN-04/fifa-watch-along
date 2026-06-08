from django.contrib import admin
from .models import (Team, Player, Match, MatchEvent,
                     WinProbabilitySnapshot, Standing, MatchConfig)


@admin.register(Team)
class TeamAdmin(admin.ModelAdmin):
    list_display = ['name', 'short_name', 'group', 'pre_match_elo', 'fc26_overall']
    list_editable = ['pre_match_elo', 'group']
    search_fields = ['name']


@admin.register(Player)
class PlayerAdmin(admin.ModelAdmin):
    list_display = ['name', 'team', 'position', 'overall_rating',
                    'pace', 'shooting', 'passing']
    list_filter = ['team', 'position']
    search_fields = ['name', 'team__name']


@admin.register(Match)
class MatchAdmin(admin.ModelAdmin):
    list_display = ['__str__', 'stage', 'status',
                    'home_score', 'away_score', 'kickoff_utc']
    list_filter = ['stage', 'status']


@admin.register(MatchEvent)
class MatchEventAdmin(admin.ModelAdmin):
    list_display = ['match', 'minute', 'event_type', 'player_name', 'assist_name']
    list_filter = ['event_type', 'match']
    ordering = ['-match', 'minute']


@admin.register(WinProbabilitySnapshot)
class WinProbAdmin(admin.ModelAdmin):
    list_display = ['match', 'minute', 'home_win_prob', 'draw_prob', 'away_win_prob']
    list_filter = ['match']


@admin.register(Standing)
class StandingAdmin(admin.ModelAdmin):
    list_display = ['group', 'position', 'team', 'played', 'points']
    list_editable = ['position', 'points']


@admin.register(MatchConfig)
class MatchConfigAdmin(admin.ModelAdmin):
    list_display = ['current_match', 'updated_at']
