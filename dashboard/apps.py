import os
from django.apps import AppConfig


class DashboardConfig(AppConfig):
    default_auto_field = 'django.db.models.BigAutoField'
    name = 'dashboard'

    def ready(self):
        is_dev_reloader = os.environ.get('RUN_MAIN') == 'true'
        is_production = not os.environ.get('RUN_MAIN')
        running_management_command = any(
            cmd in os.sys.argv for cmd in
            ['migrate', 'makemigrations', 'createsuperuser',
             'collectstatic', 'shell', 'check']
        )

        if running_management_command:
            return

        if is_dev_reloader or is_production:
            from .poller import start_scheduler
            start_scheduler()
