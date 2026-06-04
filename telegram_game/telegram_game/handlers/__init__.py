"""
handlers package
Routers live here and are included by the FastAPI app.
"""
from __future__ import annotations

from .tgwt import router as tgwt_router
from .withdraw import router as withdraw_router

__all__ = ["tgwt_router", "withdraw_router"]
