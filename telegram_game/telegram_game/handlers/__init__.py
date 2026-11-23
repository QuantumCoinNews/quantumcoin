"""
handlers package
Compatibility imports only. Retired routers are intentionally empty; the
canonical app owns authentication and ledger routes in telegram_game.runtime.
"""
from __future__ import annotations

from .tgwt import router as tgwt_router
from .withdraw import router as withdraw_router

__all__ = ["tgwt_router", "withdraw_router"]
