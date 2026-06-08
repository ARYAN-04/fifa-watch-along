from django.db import models


class Team(models.Model):
    id = models.IntegerField(primary_key=True)
    name = models.CharField(max_length=100)
    short_name = models.CharField(max_length=10, blank=True)
    flag_url = models.URLField(blank=True)
    group = models.CharField(max_length=2, blank=True)
    pre_match_elo = models.FloatField(default=1500.0)
    fc26_overall = models.IntegerField(null=True, blank=True)

    def __str__(self):
        return self.name

    class Meta:
        ordering = ['name']


class Player(models.Model):
    id = models.IntegerField(primary_key=True)
    team = models.ForeignKey(Team, on_delete=models.CASCADE,
                             related_name='players')
    name = models.CharField(max_length=100)
    position = models.CharField(max_length=10, blank=True)
    overall_rating = models.IntegerField(default=0)
    pace = models.IntegerField(default=0)
    shooting = models.IntegerField(default=0)
    passing = models.IntegerField(default=0)
    dribbling = models.IntegerField(default=0)
    defending = models.IntegerField(default=0)
    physical = models.IntegerField(default=0)
    skill_moves = models.IntegerField(default=0)
    weak_foot = models.IntegerField(default=0)
    nationality = models.CharField(max_length=50, blank=True)

    def __str__(self):
        return f"{self.name} ({self.team.name})"

    class Meta:
        ordering = ['-overall_rating']


class Match(models.Model):
    STATUS_CHOICES = [
        ('SCHEDULED', 'Scheduled'),
        ('IN_PLAY', 'In Play'),
        ('PAUSED', 'Paused'),
        ('FINISHED', 'Finished'),
        ('POSTPONED', 'Postponed'),
    ]

    id = models.IntegerField(primary_key=True)
    home_team = models.ForeignKey(Team, related_name='home_matches',
                                  on_delete=models.CASCADE)
    away_team = models.ForeignKey(Team, related_name='away_matches',
                                  on_delete=models.CASCADE)
    kickoff_utc = models.DateTimeField()
    stage = models.CharField(max_length=50)
    venue = models.CharField(max_length=100, blank=True)
    status = models.CharField(max_length=20, choices=STATUS_CHOICES,
                              default='SCHEDULED')
    home_score = models.IntegerField(default=0)
    away_score = models.IntegerField(default=0)

    def __str__(self):
        return (f"{self.home_team.short_name} vs "
                f"{self.away_team.short_name} — {self.kickoff_utc:%b %d}")

    class Meta:
        ordering = ['kickoff_utc']


class MatchEvent(models.Model):
    EVENT_CHOICES = [
        ('GOAL', 'Goal'),
        ('YELLOW_CARD', 'Yellow Card'),
        ('RED_CARD', 'Red Card'),
        ('SUBSTITUTION', 'Substitution'),
    ]

    match = models.ForeignKey(Match, on_delete=models.CASCADE,
                              related_name='events')
    minute = models.IntegerField()
    event_type = models.CharField(max_length=20, choices=EVENT_CHOICES)
    team = models.ForeignKey(Team, on_delete=models.SET_NULL, null=True)
    player_name = models.CharField(max_length=100)
    assist_name = models.CharField(max_length=100, blank=True)
    detail = models.CharField(max_length=50, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return f"{self.event_type} {self.minute}' — {self.player_name}"

    class Meta:
        ordering = ['minute']
        unique_together = ['match', 'minute', 'event_type', 'player_name']


class WinProbabilitySnapshot(models.Model):
    match = models.ForeignKey(Match, on_delete=models.CASCADE,
                              related_name='win_prob_snapshots')
    minute = models.IntegerField()
    home_win_prob = models.FloatField()
    draw_prob = models.FloatField()
    away_win_prob = models.FloatField()
    score_diff = models.IntegerField()
    xg_diff_approx = models.FloatField()
    created_at = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return (f"{self.match} min {self.minute} — "
                f"HW:{self.home_win_prob:.2f}")

    class Meta:
        ordering = ['minute']


class Standing(models.Model):
    team = models.ForeignKey(Team, on_delete=models.CASCADE)
    group = models.CharField(max_length=2)
    position = models.IntegerField()
    played = models.IntegerField(default=0)
    won = models.IntegerField(default=0)
    drawn = models.IntegerField(default=0)
    lost = models.IntegerField(default=0)
    goals_for = models.IntegerField(default=0)
    goals_against = models.IntegerField(default=0)
    points = models.IntegerField(default=0)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return f"Group {self.group} P{self.position}: {self.team.name}"

    class Meta:
        ordering = ['group', 'position']
        unique_together = ['team', 'group']


class MatchConfig(models.Model):
    current_match = models.ForeignKey(
        Match, on_delete=models.SET_NULL, null=True, blank=True,
        help_text="The match to poll live. Set this ~30 min before kickoff."
    )
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return f"Config — current match: {self.current_match}"

    class Meta:
        verbose_name = "Match Config"
        verbose_name_plural = "Match Config"
