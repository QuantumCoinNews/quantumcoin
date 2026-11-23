"""Server-side Telegram membership proof for one-time TGWT social rewards."""
from __future__ import annotations

import hashlib
import json
import os
import re
import time
from contextlib import closing

import httpx
from cryptography.hazmat.primitives import serialization

from . import reward_evidence as rewards
from .social_provider_security import reward_signing_key
from .social_reward_activation import social_rewards_enabled

PROVIDER = "TELEGRAM"
ACTION = "SOCIAL_JOIN"


def configuration():
    token = (os.getenv("TELEGRAM_BOT_TOKEN") or os.getenv("QC_TELEGRAM_BOT_TOKEN")
             or os.getenv("TELEGRAM_GAME_BOT_TOKEN") or "").strip()
    signer = reward_signing_key()
    if not re.fullmatch(r"[0-9]{5,15}:[A-Za-z0-9_-]{20,128}", token) or signer is None:
        return None
    return {"botToken": token, "signingKey": signer}


def _telegram_user_id(actor: str):
    text = str(actor or "")
    if not text.startswith("tg_"):
        return None
    raw = text[3:]
    if not re.fullmatch(r"[1-9][0-9]{0,19}", raw):
        return None
    value = int(raw)
    return value if value <= 2**63 - 1 else None


def _campaign(db, campaign_id, config, now):
    with closing(db()) as conn:
        row = conn.execute(
            "SELECT policy_json FROM tgwt_campaigns_v1 WHERE campaign_id=? AND enabled=1",
            (campaign_id,),
        ).fetchone()
    if not row:
        return None
    try:
        policy = json.loads(row[0])
        key = config["signingKey"].public_key().public_bytes(
            serialization.Encoding.Raw, serialization.PublicFormat.Raw
        ).hex()
    except Exception:
        return None
    if (policy.get("provider") != PROVIDER or policy.get("action") != ACTION
            or policy.get("publicKey") != key
            or not policy.get("startsAt", 0) <= now <= policy.get("endsAt", 0)
            or policy.get("reward_u") != rewards.TGWT_TASK_REWARD_U):
        return None
    target = policy.get("target", "")
    if not (re.fullmatch(r"-100[0-9]{5,18}", target)
            or re.fullmatch(r"@[A-Za-z][A-Za-z0-9_]{4,31}", target)):
        return None
    return policy


async def _member_status(config, policy, user_id, admit):
    if not admit():
        raise ValueError("provider_request_budget_exhausted")
    target = policy["target"]
    chat_id = int(target) if re.fullmatch(r"-100[0-9]{5,18}", target) else target
    url = "https://api.telegram.org/bot" + config["botToken"] + "/getChatMember"
    async with httpx.AsyncClient(timeout=httpx.Timeout(8), follow_redirects=False) as client:
        response = await client.get(url, params={"chat_id": chat_id, "user_id": user_id})
        response.raise_for_status()
        if len(response.content) > 262144:
            raise ValueError("provider_response_too_large")
        body = response.json()
    if not isinstance(body, dict) or body.get("ok") is not True:
        raise ValueError("provider_membership_unavailable")
    result = body.get("result")
    if not isinstance(result, dict) or not isinstance(result.get("user"), dict):
        raise ValueError("provider_membership_incomplete")
    if type(result["user"].get("id")) is not int or result["user"]["id"] != user_id:
        raise ValueError("provider_identity_mismatch")
    status = result.get("status")
    return status in {"creator", "administrator", "member"} or (
        status == "restricted" and result.get("is_member") is True
    )


async def verify(db, actor, campaign_id, now):
    if not social_rewards_enabled():
        return {"ok": False, "error": "social_rewards_disabled_mainnet_only"}
    config = configuration()
    user_id = _telegram_user_id(actor)
    if not config or user_id is None:
        return {"ok": False, "error": "telegram_verification_not_configured"}
    policy = _campaign(db, campaign_id, config, now)
    if not policy:
        return {"ok": False, "error": "telegram_campaign_not_ready"}
    try:
        with closing(db()) as conn:
            old = conn.execute(
                "SELECT reward_id FROM tgwt_verified_rewards_v1 WHERE campaign_id=? AND player_id=?",
                (campaign_id, actor),
            ).fetchone()
        if old:
            return {"ok": True, "alreadyCredited": True, "rewardId": old[0], "credited_u": 0}
        from .durable_limits import admit
        allowed = lambda: admit(db, [("TELEGRAM:provider-requests", "server-global", 30, 60)], int(time.time()))
        if not await _member_status(config, policy, user_id, allowed):
            return {"ok": False, "error": "telegram_membership_not_verified"}
    except Exception:
        # Never expose bot token, provider URL/body, or provider diagnostics.
        return {"ok": False, "error": "telegram_verification_unavailable_no_reward"}

    event_id = hashlib.sha256((str(user_id) + "\n" + ACTION + "\n" + policy["target"]).encode()).hexdigest()
    statement = {
        "version": rewards.VERSION,
        "audience": "quantumcoin.telegram-game.tgwt",
        "campaignId": policy["campaignId"],
        "campaignHash": rewards.policy_hash(policy),
        "playerId": actor,
        "provider": PROVIDER,
        "subjectId": str(user_id),
        "action": ACTION,
        "target": policy["target"],
        "eventId": event_id,
        "issuedAt": now,
        "expiresAt": now + 300,
    }
    proof = {"statement": statement, "signature": config["signingKey"].sign(rewards.signing_bytes(statement)).hex()}
    return rewards.claim(db, actor, campaign_id, proof, now, {ACTION})
