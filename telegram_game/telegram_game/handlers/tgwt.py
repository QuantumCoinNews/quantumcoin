"""
handlers/tgwt.py
TGWT endpoints router.
"""
from __future__ import annotations

from typing import Any, Optional, Dict
from fastapi import APIRouter, Query
from pydantic import BaseModel, Field

from database.store import store, safe_user_id, to_micro_units

router = APIRouter(prefix="/api/v1/tgwt", tags=["tgwt"])


class TGWTEarnReq(BaseModel):
    userId: str = Field(..., min_length=2, max_length=128)
    amount: Any
    nonce: str = Field(..., min_length=6, max_length=128)
    meta: Optional[Dict[str, Any]] = None


class TGWTVerifyReq(BaseModel):
    userId: str = Field(..., min_length=2, max_length=128)
    amount: Optional[Any] = None  # if omitted/None => verify ALL earned
    nonce: str = Field(..., min_length=6, max_length=128)
    source: Optional[str] = None


@router.get("/ping")
def ping():
    return {"ok": True, "name": "tgwt"}


@router.get("/state")
def tgwt_state(user_id: str = Query(..., alias="user_id")):
    try:
        uid = safe_user_id(user_id)
        return store.get_state(uid)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "state_failed", "detail": str(e)}


@router.post("/earn")
def tgwt_earn(req: TGWTEarnReq):
    try:
        uid = safe_user_id(req.userId)
        amt_u = to_micro_units(req.amount)
        return store.earn(uid, amt_u, req.nonce, req.meta)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "earn_failed", "detail": str(e)}


@router.post("/verify")
def tgwt_verify(req: TGWTVerifyReq):
    try:
        uid = safe_user_id(req.userId)
        amt_u = None if req.amount is None else to_micro_units(req.amount)
        return store.verify(uid, amt_u, req.nonce, req.source)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "verify_failed", "detail": str(e)}
