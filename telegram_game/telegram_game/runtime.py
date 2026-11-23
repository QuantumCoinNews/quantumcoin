"""Canonical ASGI composition root. Run from the outer telegram_game directory:

python -m uvicorn telegram_game.runtime:app --host 127.0.0.1 --port 8080 --no-proxy-headers

dev_api retains the existing route implementation as a compatibility module.
All entry points share this app, its access boundary, schema and reward policy.
"""
from .dev_api import app

__all__ = ["app"]
