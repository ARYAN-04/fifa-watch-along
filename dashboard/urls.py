from django.urls import path
from . import views

urlpatterns = [
    path('health/', views.health, name='health'),
    path('api/match/', views.match_state, name='match-state'),
    path('api/win-probability/', views.win_probability, name='win-probability'),
    path('api/events/', views.events, name='events'),
    path('api/standings/', views.standings, name='standings'),
    path('api/players/<int:team_id>/', views.players, name='players'),
]
