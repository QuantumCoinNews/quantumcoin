"""Read-only X OAuth verification adapter for separate follow/like campaigns.

Tokens exist only during completion. No automatic likes/follows or token storage.
"""
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import time
from contextlib import closing
from urllib.parse import urlencode, urlparse
import httpx
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives import serialization
from . import reward_evidence as rewards
from .social_reward_activation import social_rewards_enabled
from .social_provider_security import reward_signing_key, state_key

PROVIDER = "X"
CALLBACK = "/oauth/x/callback"


def ensure_schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS x_reward_oauth_v1 (
        state_hash TEXT PRIMARY KEY, player_id TEXT NOT NULL, campaign_id TEXT NOT NULL,
        encrypted_verifier BLOB NOT NULL, expires_at INTEGER NOT NULL, consumed INTEGER NOT NULL DEFAULT 0)""")
    conn.execute("CREATE INDEX IF NOT EXISTS x_reward_oauth_expiry_v1 ON x_reward_oauth_v1(expires_at)")


def configuration():
    client_id = os.getenv("QC_X_CLIENT_ID", "").strip()
    callback = os.getenv("QC_X_CALLBACK_URL", "").strip()
    signing_key = reward_signing_key()
    try:
        parsed = urlparse(callback)
        if not client_id or signing_key is None or parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password or parsed.path != CALLBACK or parsed.query or parsed.fragment:
            raise ValueError()
        encryption_key = state_key("QC_X_STATE_ENCRYPTION_KEY")
        if encryption_key is None:
            raise ValueError()
    except (ValueError, OSError):
        return None
    return {"clientId":client_id,"clientSecret":os.getenv("QC_X_CLIENT_SECRET", ""),"callback":callback,
            "signingKey":signing_key,"stateKey":encryption_key}


def campaign(conn, campaign_id, config, now):
    row = conn.execute("SELECT policy_json,policy_hash FROM tgwt_campaigns_v1 WHERE campaign_id=? AND enabled=1", (campaign_id,)).fetchone()
    if not row:
        return None
    policy = json.loads(row[0])
    key = config["signingKey"].public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw).hex()
    if (policy["provider"] != PROVIDER or policy["action"] not in {"SOCIAL_FOLLOW","SOCIAL_LIKE"}
            or not re.fullmatch(r"[1-9][0-9]{0,24}",policy["target"])
            or policy["publicKey"] != key or not policy["startsAt"] <= now <= policy["endsAt"]
            or policy["reward_u"] != rewards.TGWT_TASK_REWARD_U):
        return None
    return policy


def begin(db, actor, campaign_id, now):
    if not social_rewards_enabled():
        return {"ok":False,"error":"social_rewards_disabled_mainnet_only"}
    config = configuration()
    if not config:
        return {"ok":False,"error":"x_oauth_not_configured"}
    state, verifier = secrets.token_urlsafe(32), secrets.token_urlsafe(48)
    digest = hashlib.sha256(state.encode()).hexdigest()
    nonce = secrets.token_bytes(12)
    encrypted = nonce + AESGCM(config["stateKey"]).encrypt(nonce,verifier.encode(),digest.encode())
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        ensure_schema(conn)
        conn.execute("DELETE FROM x_reward_oauth_v1 WHERE expires_at<=?",(now,))
        policy = campaign(conn,campaign_id,config,now)
        if not policy:
            return {"ok":False,"error":"x_campaign_not_ready"}
        credited=conn.execute("SELECT reward_id FROM tgwt_verified_rewards_v1 WHERE campaign_id=? AND player_id=?",(campaign_id,actor)).fetchone()
        if credited:
            return {"ok":True,"alreadyCredited":True,"rewardId":credited[0],"credited_u":0}
        if conn.execute("SELECT COUNT(*) FROM x_reward_oauth_v1").fetchone()[0] >= 4096:
            return {"ok":False,"error":"x_oauth_capacity_reached"}
        conn.execute("INSERT INTO x_reward_oauth_v1 VALUES (?,?,?,?,?,0)",(digest,actor,campaign_id,encrypted,now+600))
    scope = "tweet.read users.read " + ("follows.read" if policy["action"] == "SOCIAL_FOLLOW" else "like.read")
    challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
    url = "https://x.com/i/oauth2/authorize?" + urlencode({"response_type":"code","client_id":config["clientId"],
        "redirect_uri":config["callback"],"scope":scope,"state":state,"code_challenge":challenge,"code_challenge_method":"S256"})
    return {"ok":True,"authorizeUrl":url,"state":state,"expiresAt":now+600}


async def request_json(client, method, path, admit, **kwargs):
    # No caller-controlled hosts, arbitrary URLs or action-writing endpoints.
    if not admit():
        raise ValueError("provider request budget reached")
    response = await client.request(method,"https://api.x.com"+path,**kwargs)
    response.raise_for_status()
    if len(response.content)>1_048_576:
        raise ValueError("provider response too large")
    data = response.json()
    if not isinstance(data,dict) or data.get("errors"):
        raise ValueError("provider response incomplete")
    return data


async def verify_provider_action(config, policy, code, verifier):
    async with httpx.AsyncClient(timeout=httpx.Timeout(10),follow_redirects=False) as client:
        fields={"grant_type":"authorization_code","code":code,"redirect_uri":config["callback"],"code_verifier":verifier}
        if config["clientSecret"]:
            auth=httpx.BasicAuth(config["clientId"],config["clientSecret"])
        else:
            auth=None;fields["client_id"]=config["clientId"]
        token = await request_json(client,"POST","/2/oauth2/token",config["admit"],data=fields,auth=auth)
        access = token.get("access_token")
        needed = {"tweet.read","users.read","follows.read" if policy["action"]=="SOCIAL_FOLLOW" else "like.read"}
        if not isinstance(access,str) or not access or not needed.issubset(set(str(token.get("scope","")).split())):
            raise ValueError("missing user access scope")
        headers={"Authorization":"Bearer "+access}
        me = await request_json(client,"GET","/2/users/me",config["admit"],headers=headers)
        subject=str(me.get("data",{}).get("id", ""))
        if not re.fullmatch(r"[1-9][0-9]{0,24}",subject):
            raise ValueError("provider identity unavailable")
        endpoint="following" if policy["action"]=="SOCIAL_FOLLOW" else "liked_tweets"
        pagination=None;seen=set()
        # Bounded API spend. An incomplete search never means verified.
        for _ in range(10):
            params={"max_results":100}
            if pagination: params["pagination_token"]=pagination
            page=await request_json(client,"GET",f"/2/users/{subject}/{endpoint}",config["admit"],params=params,headers=headers)
            rows=page.get("data",[])
            if not isinstance(rows,list):raise ValueError("invalid provider list")
            if any(isinstance(item,dict) and str(item.get("id",""))==policy["target"] for item in rows):
                return subject
            pagination=page.get("meta",{}).get("next_token")
            if not pagination: return None
            if not isinstance(pagination,str) or len(pagination)>2048 or pagination in seen:
                raise ValueError("invalid provider pagination")
            seen.add(pagination)
        raise ValueError("provider search budget exhausted")


async def complete(db, actor, state, code, now):
    if not social_rewards_enabled():
        return {"ok":False,"error":"social_rewards_disabled_mainnet_only"}
    config=configuration()
    if not config:
        return {"ok":False,"error":"x_oauth_not_configured"}
    if not isinstance(state,str) or not 32<=len(state)<=128 or not isinstance(code,str) or not 1<=len(code)<=4096:
        return {"ok":False,"error":"invalid_x_oauth_completion"}
    digest=hashlib.sha256(state.encode()).hexdigest()
    with closing(db()) as conn, conn:
        conn.execute("BEGIN IMMEDIATE")
        row=conn.execute("SELECT campaign_id,encrypted_verifier FROM x_reward_oauth_v1 WHERE state_hash=? AND player_id=? AND expires_at>? AND consumed=0",(digest,actor,now)).fetchone()
        if not row:
            return {"ok":False,"error":"x_oauth_state_invalid_expired_or_consumed"}
        policy=campaign(conn,row[0],config,now)
        conn.execute("UPDATE x_reward_oauth_v1 SET consumed=1 WHERE state_hash=?",(digest,))
    if not policy:
        return {"ok":False,"error":"x_campaign_not_ready"}
    try:
        from .durable_limits import admit
        config["admit"]=lambda:admit(db,[("X:provider-requests","server-global",30,60)],int(time.time()))
        encrypted=row[1]
        verifier=AESGCM(config["stateKey"]).decrypt(encrypted[:12],encrypted[12:],digest.encode()).decode()
        subject=await verify_provider_action(config,policy,code,verifier)
        if not subject:
            return {"ok":False,"error":"x_action_not_verified"}
    except Exception:
        # Never return provider bodies, OAuth codes, access tokens or client secrets.
        return {"ok":False,"error":"x_verification_unavailable_no_reward"}
    event=hashlib.sha256((subject+"\n"+policy["action"]+"\n"+policy["target"]).encode()).hexdigest()
    finished_at=max(now,int(time.time()))
    statement={"version":rewards.VERSION,"audience":"quantumcoin.telegram-game.tgwt","campaignId":policy["campaignId"],
        "campaignHash":rewards.policy_hash(policy),"playerId":actor,"provider":PROVIDER,"subjectId":subject,
        "action":policy["action"],"target":policy["target"],"eventId":event,"issuedAt":finished_at,"expiresAt":finished_at+300}
    proof={"statement":statement,"signature":config["signingKey"].sign(rewards.signing_bytes(statement)).hex()}
    return rewards.claim(db,actor,policy["campaignId"],proof,finished_at,{"SOCIAL_FOLLOW","SOCIAL_LIKE"})
