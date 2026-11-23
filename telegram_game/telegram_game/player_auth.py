"""Single HTTP identity boundary for the canonical Telegram game runtime.

Opaque bearer credentials are kept hashed in the same DB as game state.
No development identity, caller-supplied actor or admin default is trusted.
"""
from contextvars import ContextVar
import hashlib
import hmac
import json
import secrets
import time
from urllib.parse import parse_qsl

from starlette.responses import JSONResponse
from .admin_panel_safety_v1 import qc_admin_expected_token_v1, qc_admin_extract_token_v1

ACTOR = ContextVar("telegram_game_actor", default=None)
SESSION_HASH = ContextVar("telegram_game_session_hash", default=None)
PUBLIC = {
    ("GET", "/api/telegram/identity/status"),
    ("POST", "/api/telegram/identity/verify"),
    ("GET", "/api/store"), ("GET", "/api/social/links"),
    ("GET", "/api/watch"), ("GET", "/api/mining/mode"),
    ("GET", "/api/leaderboard/top"),
}
ACTOR_KEYS = {"playerId", "player_id", "userId", "user_id"}
OPERATOR_PATHS = {
    "/api/v1/tgwt/earn", "/api/v1/tgwt/verify", "/api/dev/reset",
    "/api/withdraw/mark-pending", "/api/withdraw/mock-confirm",
    "/api/withdraw/real-payout-precheck", "/admin/settlement",
}


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS player_auth_sessions_v1 (
        token_hash TEXT PRIMARY KEY, player_id TEXT NOT NULL,
        bot_fingerprint TEXT NOT NULL, created_at INTEGER NOT NULL,
        expires_at INTEGER NOT NULL)""")
    conn.execute("CREATE INDEX IF NOT EXISTS player_auth_expiry_v1 ON player_auth_sessions_v1(expires_at)")


def fingerprint(bot_token):
    return hashlib.sha256(bot_token.encode()).hexdigest()


def issue_session(db, player_id, bot_token, now):
    token = secrets.token_urlsafe(32)
    digest = hashlib.sha256(token.encode()).hexdigest()
    expires = now + 3600
    conn = db()
    try:
        conn.execute("BEGIN IMMEDIATE")
        ensure_schema(conn)
        conn.execute("DELETE FROM player_auth_sessions_v1 WHERE expires_at<=?", (now,))
        # Bound the number of valid credentials per player across workers.
        conn.execute("""DELETE FROM player_auth_sessions_v1 WHERE token_hash IN (
            SELECT token_hash FROM player_auth_sessions_v1 WHERE player_id=?
            ORDER BY created_at DESC, rowid DESC LIMIT -1 OFFSET 7)""", (player_id,))
        conn.execute("INSERT INTO player_auth_sessions_v1 VALUES (?,?,?,?,?)",
                     (digest, player_id, fingerprint(bot_token), now, expires))
        conn.commit()
    finally:
        conn.close()
    return {"accessToken": token, "tokenType": "Bearer", "expiresAt": expires}


def authenticate(db, token, bot_token, now):
    if not bot_token or not token or len(token) > 128:
        return None
    digest = hashlib.sha256(token.encode()).hexdigest()
    conn = db()
    try:
        row = conn.execute("""SELECT player_id FROM player_auth_sessions_v1
            WHERE token_hash=? AND expires_at>? AND bot_fingerprint=?""",
            (digest, now, fingerprint(bot_token))).fetchone()
    finally:
        conn.close()
    return (row[0], digest) if row else None


def operator_authorized(headers):
    expected = qc_admin_expected_token_v1()
    supplied = qc_admin_extract_token_v1(headers)
    return bool(expected and supplied and hmac.compare_digest(expected, supplied))


def operator_path(path):
    return path.startswith(("/api/admin/", "/api/settlement/", "/api/security/")) or path in OPERATOR_PATHS


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON key")
        result[key] = value
    return result


class PlayerIdentityBoundary:
    """Outermost ASGI guard: authenticate before legacy middleware or handlers.

    Reads bounded bodies and replays the ORIGINAL bytes; signed transaction
    envelopes and integers must never be rewritten by this boundary.
    """
    def __init__(self, app, db, bot_token):
        self.app, self.db, self.bot_token = app, db, bot_token
        self.login_hits = {}

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http":
            return await self.app(scope, receive, send)
        path, method = scope.get("path", ""), scope["method"]
        oauth_callbacks = {
            "/oauth/x/callback": "x_oauth_callback",
            "/oauth/youtube/callback": "youtube_oauth_callback",
        }
        if path in oauth_callbacks:
            # OAuth codes must not reach URL access logs or referrer headers.
            try:
                raw=scope.get("query_string",b"")
                if len(raw)>8192: raise ValueError()
                pairs=parse_qsl(raw.decode("utf-8"),keep_blank_values=True)
                if len(pairs)!=len({key for key,_ in pairs}): raise ValueError()
                data=dict(pairs)
                callback={key:data.get(key,"") for key in ("code","state","error")}
            except (ValueError,UnicodeError):
                callback={"error":"invalid_callback"}
            scope.setdefault("state",{})[oauth_callbacks[path]]=callback
            scope["query_string"]=b""
        guarded = path.startswith("/api/") or path == "/admin/settlement"
        # Normalize the optional trailing slash before making the access decision.
        policy_path = path.rstrip("/") or "/"
        if not guarded or method == "OPTIONS":
            return await self.app(scope, receive, send)

        async def reject(status, error):
            await JSONResponse({"ok": False, "error": error}, status_code=status,
                               headers={"Cache-Control": "no-store"})(scope, receive, send)

        if (method, policy_path) in PUBLIC and policy_path != "/api/telegram/identity/verify":
            return await self.app(scope, receive, send)
        headers = {key.decode("latin1").lower(): value.decode("latin1") for key, value in scope["headers"]}
        principal = None
        if operator_path(policy_path):
            if not operator_authorized(headers):
                return await reject(401, "operator_authentication_required")
        elif (method, policy_path) not in PUBLIC:
            auth = headers.get("authorization", "")
            if not auth.startswith("Bearer "):
                return await reject(401, "telegram_session_required")
            principal = authenticate(self.db, auth[7:], self.bot_token(), int(time.time()))
            if not principal:
                return await reject(401, "telegram_session_invalid_or_expired")
        else:
            # Rate limits use the transport peer, never a caller-controlled player/XFF.
            now = time.monotonic()
            self.login_hits = {key: hits for key, hits in self.login_hits.items() if hits and hits[-1] > now-60}
            peer = str((scope.get("client") or ("unknown",))[0])
            hits = [hit for hit in self.login_hits.get(peer, []) if hit > now-60]
            if len(hits) >= 12 or (peer not in self.login_hits and len(self.login_hits) >= 4096):
                return await reject(429, "identity_rate_limited")
            self.login_hits[peer] = hits + [now]

        chunks, size = [], 0
        while True:
            message = await receive()
            if message["type"] == "http.disconnect":
                return
            chunk = message.get("body", b"")
            size += len(chunk)
            if size > 262144:
                return await reject(413, "request_body_too_large")
            chunks.append(chunk)
            if not message.get("more_body", False):
                break
        body = b"".join(chunks)
        try:
            data = json.loads(body, object_pairs_hook=unique_object) if body else {}
            if not isinstance(data, dict):
                raise ValueError("object required")
            query = parse_qsl(scope.get("query_string", b"").decode("utf-8"), keep_blank_values=True)
        except (ValueError, UnicodeError):
            return await reject(422, "invalid_request")
        if principal:
            claimed = [value for key, value in query if key in ACTOR_KEYS]
            claimed.extend(value for key, value in data.items() if key in ACTOR_KEYS)
            if any(not isinstance(value, str) or value != principal[0] for value in claimed):
                return await reject(403, "player_identity_mismatch")
        actor_reset = ACTOR.set(principal[0] if principal else None)
        hash_reset = SESSION_HASH.set(principal[1] if principal else None)
        scope.setdefault("state", {})["verified_player_id"] = ACTOR.get()
        delivered = False

        async def replay():
            nonlocal delivered
            if not delivered:
                delivered = True
                return {"type": "http.request", "body": body, "more_body": False}
            return await receive()

        async def private_send(message):
            if message["type"] == "http.response.start":
                message = dict(message)
                message["headers"] = [(k, v) for k, v in message["headers"] if k.lower() != b"cache-control"] + [(b"cache-control", b"no-store")]
            await send(message)
        try:
            await self.app(scope, replay, private_send)
        finally:
            SESSION_HASH.reset(hash_reset)
            ACTOR.reset(actor_reset)
