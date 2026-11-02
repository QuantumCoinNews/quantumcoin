"""One-use wallet challenges. Public signatures only; never imports wallet keys.

Address hashing matches wallet.HashPubKey: HASH160(X || Y), not HASH160(04||X||Y).
The caller must authenticate Telegram identity before issuing/consuming a proof.
Proofs are scoped to the node-verified network/campaign/genesis and award period.
"""
import hashlib
import json
import re
import secrets
import sqlite3

from .nft_mint_bridge_v1 import _wallet_owner, NFTMintBridgeError
from .nft_program_bridge_v1 import _validate_program_v1

VERSION = "QC_NFT_WALLET_CHALLENGE_V1"
DOMAIN = b"QC_NFT_WALLET_CHALLENGE_V1\x00"


class WalletProofRejected(ValueError):
    pass


def require(ok, code):
    if not ok:
        raise WalletProofRejected(code)


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True, allow_nan=False)


def program_scope(program):
    require(_validate_program_v1(program) is not None, "verified_program_required")
    return {"networkID": program["networkID"], "campaignID": program["campaignID"],
            "legacyGenesisHex": program["legacyGenesisHex"].lower()}


def tables(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_wallet_challenges_v1 (
        nonce TEXT PRIMARY KEY, challenge_json TEXT NOT NULL,
        consumed_at INTEGER, public_key TEXT, signature TEXT,
        user_id TEXT NOT NULL, period TEXT NOT NULL)""")
    conn.execute("CREATE INDEX IF NOT EXISTS idx_nft_wallet_proofs_scope_v1 ON monthly_nft_wallet_challenges_v1(user_id,period,consumed_at)")


def linked_owner(conn, user):
    row = conn.execute("SELECT qc_address FROM players WHERE player_id=?", (user,)).fetchone()
    require(row is not None, "linked_wallet_required")
    try:
        return _wallet_owner(row[0])
    except NFTMintBridgeError:
        raise WalletProofRejected("linked_wallet_invalid") from None


def verify_signature(challenge, public_hex, signature_hex):
    # cryptography parses curve points and ASN.1; no custom elliptic-curve math.
    try:
        from cryptography.hazmat.primitives import hashes
        from cryptography.hazmat.primitives.asymmetric import ec
        from cryptography.exceptions import InvalidSignature
    except ImportError:
        raise WalletProofRejected("wallet_crypto_dependency_unavailable") from None
    require(isinstance(public_hex, str) and re.fullmatch(r"04[0-9a-f]{128}", public_hex), "invalid_wallet_public_key")
    require(isinstance(signature_hex, str) and re.fullmatch(r"[0-9a-f]{16,144}", signature_hex)
            and len(signature_hex) % 2 == 0, "invalid_wallet_signature")
    raw = bytes.fromhex(public_hex)
    owner = hashlib.new("ripemd160", hashlib.sha256(raw[1:]).digest()).hexdigest()
    require(owner == challenge["walletOwner"], "wallet_key_owner_mismatch")
    try:
        key = ec.EllipticCurvePublicKey.from_encoded_point(ec.SECP256K1(), raw)
        key.verify(bytes.fromhex(signature_hex), DOMAIN + canonical(challenge).encode("ascii"), ec.ECDSA(hashes.SHA256()))
    except (InvalidSignature, ValueError):
        raise WalletProofRejected("invalid_wallet_signature") from None
    return True


def issue_wallet_challenge_v1(connect, program, user, period, now):
    conn = None
    try:
        scope = program_scope(program)
        require(isinstance(user, str) and re.fullmatch(r"tg_[0-9]{1,20}", user), "telegram_identity_required")
        require(period in program["awardPeriods"], "wallet_period_outside_program")
        require(type(now) is int and now > 0, "invalid_wallet_challenge_time")
        conn = connect()
        conn.execute("BEGIN IMMEDIATE")
        owner = linked_owner(conn, user)
        tables(conn)
        # Reissuing does not consume/revoke a previously proven identity.
        challenge = {"version": VERSION, **scope, "userId": user, "period": period,
                     "walletOwner": owner, "nonce": secrets.token_hex(32),
                     "issuedAt": now, "expiresAt": now + 600}
        conn.execute("INSERT INTO monthly_nft_wallet_challenges_v1(nonce,challenge_json,user_id,period) VALUES(?,?,?,?)",
                     (challenge["nonce"], canonical(challenge), user, period))
        conn.commit()
        return {"ok": True, "challenge": challenge, "signingBytesHex": (DOMAIN + canonical(challenge).encode("ascii")).hex()}
    except WalletProofRejected as error:
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "wallet_proof_store_unavailable"}
    finally:
        if conn is not None:
            conn.close()


def consume_wallet_challenge_v1(connect, program, user, nonce, public_hex, signature_hex, now):
    conn = None
    try:
        scope = program_scope(program)
        require(isinstance(nonce, str) and re.fullmatch(r"[0-9a-f]{64}", nonce), "invalid_wallet_nonce")
        require(type(now) is int and now > 0, "invalid_wallet_challenge_time")
        conn = connect()
        conn.execute("BEGIN IMMEDIATE")
        tables(conn)
        row = conn.execute("SELECT challenge_json,consumed_at FROM monthly_nft_wallet_challenges_v1 WHERE nonce=?", (nonce,)).fetchone()
        require(row is not None, "wallet_challenge_not_found")
        require(row[1] is None, "wallet_challenge_replayed")
        challenge = json.loads(row[0])
        require(set(challenge) == {"version", "networkID", "campaignID", "legacyGenesisHex", "userId", "period", "walletOwner", "nonce", "issuedAt", "expiresAt"}
                and challenge["version"] == VERSION and challenge["nonce"] == nonce, "invalid_wallet_challenge")
        require(challenge["userId"] == user and all(challenge[k] == v for k, v in scope.items())
                and challenge["period"] in program["awardPeriods"], "wallet_challenge_scope_mismatch")
        require(challenge["issuedAt"] <= now < challenge["expiresAt"], "wallet_challenge_expired")
        require(linked_owner(conn, user) == challenge["walletOwner"], "wallet_binding_changed")
        verify_signature(challenge, public_hex, signature_hex)
        conn.execute("UPDATE monthly_nft_wallet_challenges_v1 SET consumed_at=?,public_key=?,signature=? WHERE nonce=? AND consumed_at IS NULL",
                     (now, public_hex, signature_hex, nonce))
        conn.commit()
        return {"ok": True, "walletOwner": challenge["walletOwner"], "period": challenge["period"]}
    except WalletProofRejected as error:
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, TypeError, KeyError):
        return {"ok": False, "error": "wallet_proof_store_unavailable"}
    finally:
        if conn is not None:
            conn.close()


def require_wallet_proof_v1(conn, program, user, period, owner, cutoff):
    """Reverify stored public proof, including its pre-freeze consumption time."""
    scope = program_scope(program)
    exists = conn.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='monthly_nft_wallet_challenges_v1'").fetchone()
    require(exists, "wallet_control_proof_required")
    rows = conn.execute("SELECT challenge_json,consumed_at,public_key,signature FROM monthly_nft_wallet_challenges_v1 WHERE user_id=? AND period=? AND consumed_at IS NOT NULL AND consumed_at<=?", (user, period, cutoff))
    for raw, consumed, public, signature in rows:
        challenge = json.loads(raw)
        if (challenge.get("userId"), challenge.get("period"), challenge.get("walletOwner")) != (user, period, owner):
            continue
        if not all(challenge.get(k) == v for k, v in scope.items()):
            continue
        require(challenge["version"] == VERSION and challenge["issuedAt"] <= consumed < challenge["expiresAt"], "wallet_control_proof_invalid")
        verify_signature(challenge, public, signature)
        require(linked_owner(conn, user) == owner, "wallet_binding_changed")
        return hashlib.sha256(raw.encode("ascii")).hexdigest()
    raise WalletProofRejected("wallet_control_proof_required")
