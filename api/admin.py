import os
from sqladmin import Admin, ModelView
from sqladmin.authentication import AuthenticationBackend
from starlette.requests import Request

from api.models import MatchConfig, Match, Team, Standing, Player, MatchEvent, WinProbabilitySnapshot

class AdminAuth(AuthenticationBackend):
    async def login(self, request: Request) -> bool:
        form = await request.form()
        username = form.get("username")
        password = form.get("password")
        
        # Simple username/password from env, defaulting to admin/admin
        admin_user = os.getenv("ADMIN_USERNAME", "admin")
        admin_pass = os.getenv("ADMIN_PASSWORD", "admin")
        
        if username == admin_user and password == admin_pass:
            request.session.update({"token": "authenticated"})
            return True
        return False

    async def logout(self, request: Request) -> bool:
        request.session.clear()
        return True

    async def authenticate(self, request: Request) -> bool:
        token = request.session.get("token")
        if token == "authenticated":
            return True
        return False

# Model Views
class MatchConfigAdmin(ModelView, model=MatchConfig):
    column_list = [MatchConfig.id, MatchConfig.current_match_id, MatchConfig.updated_at]
    form_columns = [MatchConfig.current_match_id]
    name = "Match Config"
    name_plural = "Match Configs"

class MatchAdmin(ModelView, model=Match):
    column_list = [Match.id, Match.home_team_id, Match.away_team_id, Match.status, Match.home_score, Match.away_score, Match.kickoff_utc]
    form_columns = [Match.id, Match.home_team_id, Match.away_team_id, Match.status, Match.home_score, Match.away_score, Match.kickoff_utc, Match.stage, Match.venue]

class TeamAdmin(ModelView, model=Team):
    column_list = [Team.id, Team.name, Team.short_name, Team.group, Team.pre_match_elo]
    form_columns = [Team.id, Team.name, Team.short_name, Team.flag_url, Team.group, Team.pre_match_elo, Team.fc26_overall]

class StandingAdmin(ModelView, model=Standing):
    column_list = [Standing.id, Standing.team_id, Standing.group, Standing.position, Standing.points]
    form_columns = [Standing.team_id, Standing.group, Standing.position, Standing.played, Standing.won, Standing.drawn, Standing.lost, Standing.goals_for, Standing.goals_against, Standing.points]

class PlayerAdmin(ModelView, model=Player):
    column_list = [Player.id, Player.name, Player.team_id, Player.position, Player.overall_rating]
    form_columns = [Player.id, Player.team_id, Player.name, Player.position, Player.overall_rating, Player.pace, Player.shooting, Player.passing, Player.dribbling, Player.defending, Player.physical, Player.skill_moves, Player.weak_foot, Player.nationality]

class MatchEventAdmin(ModelView, model=MatchEvent):
    column_list = [MatchEvent.id, MatchEvent.match_id, MatchEvent.minute, MatchEvent.event_type, MatchEvent.player_name]

class WinProbabilitySnapshotAdmin(ModelView, model=WinProbabilitySnapshot):
    column_list = [WinProbabilitySnapshot.id, WinProbabilitySnapshot.match_id, WinProbabilitySnapshot.minute, WinProbabilitySnapshot.home_win_prob, WinProbabilitySnapshot.draw_prob, WinProbabilitySnapshot.away_win_prob]

def setup_admin(app, engine):
    secret_key = os.getenv("SECRET_KEY", "dev-insecure-key-change-in-production")
    authentication_backend = AdminAuth(secret_key=secret_key)
    
    admin = Admin(app, engine, authentication_backend=authentication_backend)
    
    # Register views
    admin.add_view(MatchConfigAdmin)
    admin.add_view(MatchAdmin)
    admin.add_view(TeamAdmin)
    admin.add_view(StandingAdmin)
    admin.add_view(PlayerAdmin)
    admin.add_view(MatchEventAdmin)
    admin.add_view(WinProbabilitySnapshotAdmin)
    
    return admin
