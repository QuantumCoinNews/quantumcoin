"""Provider-signed, single-use task rewards. This module never sends BSC tokens.

The signing key belongs to an independent server-side verification adapter,
never the WebApp. No adapter is implicitly trusted or enabled by default.
"""
import hashlib
import json
import re
import secrets
from contextlib import closing
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
from .tgwt_pool_policy import TGWT_POOL_TOTAL_U as CAP_U
from .tgwt_pool_policy import TGWT_TASK_REWARD_U

VERSION = "QC_TGWT_PROVIDER_EVIDENCE_V1"
ACTIONS = {"SOCIAL_FOLLOW", "SOCIAL_LIKE", "SOCIAL_JOIN", "SOCIAL_SUBSCRIBE", "WATCH_COMPLETE", "STORE_PURCHASE"}
SOCIAL_ACTIONS_BY_PROVIDER = {
    "X": {"SOCIAL_FOLLOW", "SOCIAL_LIKE"},
    "TELEGRAM": {"SOCIAL_JOIN"},
    "YOUTUBE": {"SOCIAL_SUBSCRIBE"},
}
CAMPAIGN_URL_HOSTS = {
    "X": {"x.com", "www.x.com", "twitter.com", "www.twitter.com"},
    "TELEGRAM": {"t.me", "www.t.me", "web.telegram.org"},
    "YOUTUBE": {"youtube.com", "www.youtube.com", "youtu.be"},
}


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True, allow_nan=False).encode()


def policy_hash(policy):
    return hashlib.sha256(b"QC_TGWT_CAMPAIGN_V1\n" + canonical(policy)).hexdigest()


def signing_bytes(statement):
    return VERSION.encode() + b"\n" + canonical(statement)


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS tgwt_campaigns_v1 (
        campaign_id TEXT PRIMARY KEY, policy_json TEXT NOT NULL, policy_hash TEXT NOT NULL,
        spent_u INTEGER NOT NULL DEFAULT 0, enabled INTEGER NOT NULL DEFAULT 1)""")
    conn.execute("""CREATE TABLE IF NOT EXISTS tgwt_verified_rewards_v1 (
        reward_id TEXT PRIMARY KEY, campaign_id TEXT NOT NULL, player_id TEXT NOT NULL,
        provider TEXT NOT NULL, subject_id TEXT NOT NULL, event_id TEXT NOT NULL,
        amount_u INTEGER NOT NULL CHECK(amount_u>0), proof_hash TEXT NOT NULL, created_at INTEGER NOT NULL,
        UNIQUE(campaign_id,player_id), UNIQUE(campaign_id,subject_id), UNIQUE(provider,event_id))""")


def create_campaign(db, policy):
    expected = {"campaignId", "provider", "publicKey", "action", "target", "reward_u", "budget_u", "startsAt", "endsAt"}
    try:
        if set(policy) not in (expected, expected | {"url"}) or not re.fullmatch(r"[a-zA-Z0-9_-]{1,64}", policy["campaignId"]):
            raise ValueError()
        if policy["action"] not in ACTIONS or not all(isinstance(policy[key], str) and 0 < len(policy[key]) <= 256 for key in ("provider", "target")):
            raise ValueError()
        provider = policy["provider"].upper()
        if provider in SOCIAL_ACTIONS_BY_PROVIDER and policy["action"] not in SOCIAL_ACTIONS_BY_PROVIDER[provider]:
            raise ValueError()
        if "url" in policy:
            from urllib.parse import urlsplit
            url = policy["url"]
            parsed = urlsplit(url) if isinstance(url, str) else None
            if (not parsed or parsed.scheme != "https" or parsed.hostname not in CAMPAIGN_URL_HOSTS.get(provider, set())
                    or parsed.username or parsed.password or parsed.port or parsed.query):
                raise ValueError()
        elif provider in {"TELEGRAM", "YOUTUBE"}:
            # New provider tasks must expose the exact destination to the player.
            raise ValueError()
        if provider == "X" and policy["action"] in SOCIAL_ACTIONS_BY_PROVIDER[provider]:
            if not re.fullmatch(r"[1-9][0-9]{0,24}", policy["target"]):
                raise ValueError()
        if provider == "TELEGRAM":
            if not (re.fullmatch(r"-100[0-9]{5,18}", policy["target"])
                    or re.fullmatch(r"@[A-Za-z][A-Za-z0-9_]{4,31}", policy["target"])):
                raise ValueError()
        if provider == "YOUTUBE" and not (
                re.fullmatch(r"UC[A-Za-z0-9_-]{22}", policy["target"])
                or re.fullmatch(r"@?[A-Za-z0-9._-]{3,30}", policy["target"])):
            raise ValueError()
        if any(type(policy[key]) is not int for key in ("reward_u", "budget_u", "startsAt", "endsAt")):
            raise ValueError()
        if not policy["reward_u"] == TGWT_TASK_REWARD_U <= policy["budget_u"] <= CAP_U or not 0 < policy["startsAt"] < policy["endsAt"]:
            raise ValueError()
        key = bytes.fromhex(policy["publicKey"])
        if len(key) != 32:
            raise ValueError()
        Ed25519PublicKey.from_public_bytes(key)
    except (ValueError, TypeError, KeyError):
        return {"ok": False, "error": "invalid_campaign_policy"}
    digest = policy_hash(policy)
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        ensure_schema(conn)
        old = conn.execute("SELECT policy_hash FROM tgwt_campaigns_v1 WHERE campaign_id=?", (policy["campaignId"],)).fetchone()
        if old:
            return {"ok": old[0] == digest, "error": None if old[0] == digest else "campaign_policy_immutable", "campaignHash": old[0]}
        conn.execute("INSERT INTO tgwt_campaigns_v1(campaign_id,policy_json,policy_hash) VALUES (?,?,?)",
                     (policy["campaignId"], canonical(policy).decode(), digest))
    return {"ok": True, "campaignId": policy["campaignId"], "campaignHash": digest}


def claim(db, player_id, campaign_id, proof, now, allowed_actions=None):
    if not campaign_id or not isinstance(proof, dict):
        return {"ok": False, "error": "verified_provider_evidence_required"}
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        row = conn.execute("SELECT policy_json,policy_hash,spent_u,enabled FROM tgwt_campaigns_v1 WHERE campaign_id=?", (campaign_id,)).fetchone()
        if not row or not row[3]:
            return {"ok": False, "error": "campaign_unavailable"}
        policy = json.loads(row[0])
        if policy["reward_u"] != TGWT_TASK_REWARD_U:
            return {"ok": False, "error": "campaign_reward_policy_mismatch"}
        if not policy["startsAt"] <= now <= policy["endsAt"] or (allowed_actions and policy["action"] not in allowed_actions):
            return {"ok": False, "error": "campaign_action_or_period_mismatch"}
        try:
            if set(proof) != {"statement", "signature"}:
                raise ValueError()
            statement = proof["statement"]
            if set(statement) != {"version", "audience", "campaignId", "campaignHash", "playerId", "provider", "subjectId", "action", "target", "eventId", "issuedAt", "expiresAt"}:
                raise ValueError()
            for key, expected in (("version", VERSION), ("audience", "quantumcoin.telegram-game.tgwt"),
                                  ("campaignId", campaign_id), ("campaignHash", row[1]), ("playerId", player_id),
                                  ("provider", policy["provider"]), ("action", policy["action"]), ("target", policy["target"])):
                if statement[key] != expected:
                    raise ValueError()
            if any(type(statement[key]) is not int for key in ("issuedAt", "expiresAt")):
                raise ValueError()
            if not now-600 <= statement["issuedAt"] <= now+30 or not now <= statement["expiresAt"] <= statement["issuedAt"]+600:
                raise ValueError()
            if not all(isinstance(statement[key], str) and 0 < len(statement[key]) <= 256 for key in ("subjectId", "eventId")):
                raise ValueError()
            Ed25519PublicKey.from_public_bytes(bytes.fromhex(policy["publicKey"])).verify(bytes.fromhex(proof["signature"]), signing_bytes(statement))
        except Exception:
            return {"ok": False, "error": "provider_evidence_invalid"}
        old = conn.execute("SELECT reward_id FROM tgwt_verified_rewards_v1 WHERE campaign_id=? AND player_id=?", (campaign_id, player_id)).fetchone()
        if old:
            return {"ok": True, "alreadyCredited": True, "rewardId": old[0], "credited_u": 0, "onChainPaid": False}
        if conn.execute("SELECT 1 FROM tgwt_verified_rewards_v1 WHERE (campaign_id=? AND subject_id=?) OR (provider=? AND event_id=?)",
                        (campaign_id, statement["subjectId"], policy["provider"], statement["eventId"])).fetchone():
            return {"ok": False, "error": "provider_account_or_event_already_rewarded"}
        amount = policy["reward_u"]
        if row[2] > policy["budget_u"]-amount:
            return {"ok": False, "error": "campaign_budget_exhausted"}
        updated = conn.execute("""UPDATE tgwt_pool SET distributed_u=distributed_u+?,updated_at=?
            WHERE id=1 AND total_u<=? AND distributed_u>=0 AND distributed_u<=total_u-?""", (amount, now, CAP_U, amount)).rowcount
        if updated != 1:
            return {"ok": False, "error": "pool_exhausted_or_invalid"}
        reward_id = "tgwr_" + secrets.token_hex(16)
        conn.execute("INSERT OR IGNORE INTO tgwt_state(user_id,updated_at) VALUES (?,?)", (player_id, now))
        conn.execute("""UPDATE tgwt_state SET earned_u=earned_u+?,verified_u=verified_u+?,rev=rev+1,updated_at=? WHERE user_id=?""", (amount, amount, now, player_id))
        conn.execute("UPDATE tgwt_campaigns_v1 SET spent_u=spent_u+? WHERE campaign_id=?", (amount, campaign_id))
        conn.execute("INSERT INTO tgwt_verified_rewards_v1 VALUES (?,?,?,?,?,?,?,?,?)", (reward_id, campaign_id, player_id,
                     policy["provider"], statement["subjectId"], statement["eventId"], amount, hashlib.sha256(canonical(proof)).hexdigest(), now))
        conn.execute("INSERT INTO tgwt_events(id,user_id,kind,amount_u,meta_json,created_at) VALUES (?,?,?,?,?,?)",
                     (reward_id, player_id, "VERIFIED_TASK_REWARD", amount, json.dumps({"campaignId":campaign_id,"action":policy["action"],"onChainPaid":False}), now))
        return {"ok": True, "alreadyCredited": False, "rewardId": reward_id, "credited_u": amount,
                "status": "VERIFIED_OFFCHAIN_CLAIMABLE", "onChainPaid": False, "gasPayer": "user", "gasAsset": "BNB"}
