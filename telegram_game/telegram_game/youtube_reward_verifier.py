"""Read-only YouTube OAuth proof for one-time TGWT subscription rewards."""
from __future__ import annotations

import base64
import hashlib
import json
import os
import re
import secrets
import time
from contextlib import closing
from urllib.parse import urlencode

import httpx
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

from . import reward_evidence as rewards
from .social_provider_security import https_callback, reward_signing_key, state_key
from .social_reward_activation import social_rewards_enabled

PROVIDER = "YOUTUBE"
ACTION = "SOCIAL_SUBSCRIBE"
CALLBACK = "/oauth/youtube/callback"
SCOPES = "openid https://www.googleapis.com/auth/youtube.readonly"


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS youtube_reward_oauth_v1 (
        state_hash TEXT PRIMARY KEY, player_id TEXT NOT NULL, campaign_id TEXT NOT NULL,
        encrypted_verifier BLOB NOT NULL, expires_at INTEGER NOT NULL,
        consumed INTEGER NOT NULL DEFAULT 0)""")
    conn.execute("CREATE INDEX IF NOT EXISTS youtube_reward_oauth_expiry_v1 ON youtube_reward_oauth_v1(expires_at)")


def configuration():
    client_id = os.getenv("QC_YOUTUBE_CLIENT_ID", "").strip()
    callback = https_callback(os.getenv("QC_YOUTUBE_CALLBACK_URL", ""), CALLBACK)
    client_secret = os.getenv("QC_YOUTUBE_CLIENT_SECRET", "").strip()
    signer = reward_signing_key()
    key = state_key("QC_YOUTUBE_STATE_ENCRYPTION_KEY")
    if not client_id or not callback or not signer or not key:
        return None
    if len(client_id) > 256 or len(client_secret) > 512:
        return None
    return {"clientId": client_id, "clientSecret": client_secret, "callback": callback,
            "signingKey": signer, "stateKey": key}


def campaign(db, campaign_id, config, now, conn=None):
    if conn is not None:
        row = conn.execute("SELECT policy_json FROM tgwt_campaigns_v1 WHERE campaign_id=? AND enabled=1", (campaign_id,)).fetchone()
    else:
        with closing(db()) as read_conn:
            row = read_conn.execute("SELECT policy_json FROM tgwt_campaigns_v1 WHERE campaign_id=? AND enabled=1", (campaign_id,)).fetchone()
    if not row:
        return None
    try:
        policy = json.loads(row[0])
        from cryptography.hazmat.primitives import serialization
        public_key = config["signingKey"].public_key().public_bytes(
            serialization.Encoding.Raw, serialization.PublicFormat.Raw
        ).hex()
    except Exception:
        return None
    target = policy.get("target", "")
    if (policy.get("provider") != PROVIDER or policy.get("action") != ACTION
            or policy.get("publicKey") != public_key
            or policy.get("reward_u") != rewards.TGWT_TASK_REWARD_U
            or not policy.get("startsAt", 0) <= now <= policy.get("endsAt", 0)
            or not (re.fullmatch(r"UC[A-Za-z0-9_-]{22}", target)
                    or re.fullmatch(r"@?[A-Za-z0-9._-]{3,30}", target))):
        return None
    return policy


def begin(db, actor, campaign_id, now):
    if not social_rewards_enabled():
        return {"ok": False, "error": "social_rewards_disabled_mainnet_only"}
    config = configuration()
    if not config:
        return {"ok": False, "error": "youtube_oauth_not_configured"}
    state, verifier = secrets.token_urlsafe(32), secrets.token_urlsafe(48)
    digest = hashlib.sha256(state.encode()).hexdigest()
    nonce = secrets.token_bytes(12)
    encrypted = nonce + AESGCM(config["stateKey"]).encrypt(nonce, verifier.encode(), digest.encode())
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        ensure_schema(conn)
        conn.execute("DELETE FROM youtube_reward_oauth_v1 WHERE expires_at<=?", (now,))
        policy = campaign(db, campaign_id, config, now, conn)
        if not policy:
            return {"ok": False, "error": "youtube_campaign_not_ready"}
        credited = conn.execute(
            "SELECT reward_id FROM tgwt_verified_rewards_v1 WHERE campaign_id=? AND player_id=?",
            (campaign_id, actor),
        ).fetchone()
        if credited:
            return {"ok": True, "alreadyCredited": True, "rewardId": credited[0], "credited_u": 0}
        if conn.execute("SELECT COUNT(*) FROM youtube_reward_oauth_v1").fetchone()[0] >= 4096:
            return {"ok": False, "error": "youtube_oauth_capacity_reached"}
        conn.execute("INSERT INTO youtube_reward_oauth_v1 VALUES (?,?,?,?,?,0)",
                     (digest, actor, campaign_id, encrypted, now + 600))
    challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
    authorize_url = "https://accounts.google.com/o/oauth2/v2/auth?" + urlencode({
        "response_type": "code", "client_id": config["clientId"], "redirect_uri": config["callback"],
        "scope": SCOPES, "state": state, "code_challenge": challenge,
        "code_challenge_method": "S256", "access_type": "online",
    })
    return {"ok": True, "authorizeUrl": authorize_url, "state": state, "expiresAt": now + 600}


async def _request_json(client, method, url, admit, **kwargs):
    if not admit():
        raise ValueError("provider request budget reached")
    response = await client.request(method, url, **kwargs)
    response.raise_for_status()
    if len(response.content) > 1_048_576:
        raise ValueError("provider response too large")
    data = response.json()
    if not isinstance(data, dict) or data.get("error"):
        raise ValueError("provider response incomplete")
    return data


async def _verify_subscription(config, policy, code, verifier, admit):
    async with httpx.AsyncClient(timeout=httpx.Timeout(10), follow_redirects=False) as client:
        fields = {"grant_type": "authorization_code", "code": code,
                  "redirect_uri": config["callback"], "client_id": config["clientId"],
                  "code_verifier": verifier}
        if config["clientSecret"]:
            fields["client_secret"] = config["clientSecret"]
        token = await _request_json(client, "POST", "https://oauth2.googleapis.com/token", admit, data=fields)
        access_token = token.get("access_token")
        if not isinstance(access_token, str) or not access_token or len(access_token) > 8192:
            raise ValueError("provider token unavailable")
        headers = {"Authorization": "Bearer " + access_token}
        identity = await _request_json(client, "GET", "https://openidconnect.googleapis.com/v1/userinfo", admit, headers=headers)
        subject = identity.get("sub")
        if not isinstance(subject, str) or not re.fullmatch(r"[A-Za-z0-9_-]{8,255}", subject):
            raise ValueError("provider identity unavailable")

        target = policy["target"]
        if target.startswith("UC"):
            channel_id = target
        else:
            handle = target.lstrip("@")
            channels = await _request_json(client, "GET", "https://www.googleapis.com/youtube/v3/channels",
                admit, params={"part": "id", "forHandle": handle}, headers=headers)
            items = channels.get("items")
            if not isinstance(items, list) or len(items) != 1 or not isinstance(items[0], dict):
                raise ValueError("target_channel_not_resolved")
            channel_id = items[0].get("id")
            if not isinstance(channel_id, str) or not re.fullmatch(r"UC[A-Za-z0-9_-]{22}", channel_id):
                raise ValueError("target_channel_invalid")

        subscriptions = await _request_json(client, "GET", "https://www.googleapis.com/youtube/v3/subscriptions",
            admit, params={"part": "snippet", "mine": "true", "forChannelId": channel_id, "maxResults": 50},
            headers=headers)
        items = subscriptions.get("items", [])
        if not isinstance(items, list):
            raise ValueError("subscription_response_invalid")
        subscribed = any(
            isinstance(item, dict)
            and isinstance(item.get("snippet"), dict)
            and isinstance(item["snippet"].get("resourceId"), dict)
            and item["snippet"]["resourceId"].get("channelId") == channel_id
            for item in items
        )
        return subject if subscribed else None


async def complete(db, actor, state, code, now):
    if not social_rewards_enabled():
        return {"ok": False, "error": "social_rewards_disabled_mainnet_only"}
    config = configuration()
    if not config:
        return {"ok": False, "error": "youtube_oauth_not_configured"}
    if not isinstance(state, str) or not 32 <= len(state) <= 128 or not isinstance(code, str) or not 1 <= len(code) <= 4096:
        return {"ok": False, "error": "invalid_youtube_oauth_completion"}
    digest = hashlib.sha256(state.encode()).hexdigest()
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        row = conn.execute("""SELECT campaign_id,encrypted_verifier FROM youtube_reward_oauth_v1
            WHERE state_hash=? AND player_id=? AND expires_at>? AND consumed=0""", (digest, actor, now)).fetchone()
        if not row:
            return {"ok": False, "error": "youtube_oauth_state_invalid_expired_or_consumed"}
        policy = campaign(db, row[0], config, now, conn)
        conn.execute("UPDATE youtube_reward_oauth_v1 SET consumed=1 WHERE state_hash=?", (digest,))
    if not policy:
        return {"ok": False, "error": "youtube_campaign_not_ready"}
    try:
        from .durable_limits import admit
        allow = lambda: admit(db, [("YOUTUBE:provider-requests", "server-global", 30, 60)], int(time.time()))
        encrypted = row[1]
        verifier = AESGCM(config["stateKey"]).decrypt(encrypted[:12], encrypted[12:], digest.encode()).decode()
        subject = await _verify_subscription(config, policy, code, verifier, allow)
        if not subject:
            return {"ok": False, "error": "youtube_subscription_not_verified"}
    except Exception:
        # Never return provider bodies, OAuth codes, access tokens or client secrets.
        return {"ok": False, "error": "youtube_verification_unavailable_no_reward"}

    event_id = hashlib.sha256((subject + "\n" + ACTION + "\n" + policy["target"]).encode()).hexdigest()
    issued = max(now, int(time.time()))
    statement = {"version": rewards.VERSION, "audience": "quantumcoin.telegram-game.tgwt",
        "campaignId": policy["campaignId"], "campaignHash": rewards.policy_hash(policy),
        "playerId": actor, "provider": PROVIDER, "subjectId": subject,
        "action": ACTION, "target": policy["target"], "eventId": event_id,
        "issuedAt": issued, "expiresAt": issued + 300}
    proof = {"statement": statement, "signature": config["signingKey"].sign(rewards.signing_bytes(statement)).hex()}
    return rewards.claim(db, actor, policy["campaignId"], proof, issued, {ACTION})
