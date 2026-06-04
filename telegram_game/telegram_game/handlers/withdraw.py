"""
handlers/withdraw.py
Withdraw endpoints router.
- POST /api/v1/withdraw/request
- GET  /api/v1/withdraw/status?id=...
- GET  /api/v1/withdraw/list?user_id=...&limit=...
"""
from __future__ import annotations

from typing import Any
from fastapi import APIRouter, Query
from pydantic import BaseModel, Field

from database.store import store, safe_user_id, to_micro_units

router = APIRouter(prefix="/api/v1/withdraw", tags=["withdraw"])


class WithdrawReq(BaseModel):
    userId: str = Field(..., min_length=2, max_length=128)
    asset: str = Field(..., min_length=2, max_length=16)  # "TGWT"
    amount: Any
    toAddress: str = Field(..., min_length=8, max_length=128)
    nonce: str = Field(..., min_length=6, max_length=128)


@router.get("/ping")
def ping():
    return {"ok": True, "name": "withdraw"}


@router.post("/request")
def withdraw_request(req: WithdrawReq):
    try:
        user_id = safe_user_id(req.userId)
        amt_u = to_micro_units(req.amount)
        return store.create_withdraw_request(user_id, req.asset, amt_u, req.toAddress, req.nonce)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "withdraw_request_failed", "detail": str(e)}


@router.get("/status")
def withdraw_status(id: str = Query(...)):
    try:
        return store.get_withdraw_status(id)
    except Exception as e:
        return {"ok": False, "error": "status_failed", "detail": str(e)}


@router.get("/list")
def withdraw_list(user_id: str = Query(..., alias="user_id"), limit: int = 20):
    try:
        uid = safe_user_id(user_id)
        return store.list_withdraws(uid, limit=limit)
    except ValueError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": "list_failed", "detail": str(e)}
