"""Wallet control proofs, independent of the monthly NFT award protocol.

Keys never cross HTTP. Legacy address associations remain historical hints until
the wallet holder proves control; they cannot reserve another person's address.
"""
import hashlib
import json
import os
import re
import secrets
from contextlib import closing
from .utils.nft_mint_bridge_v1 import _wallet_owner
from .utils.nft_wallet_proof_v1 import canonical, WalletProofRejected

DOMAIN = b"QC_GAME_WALLET_LINK_V1\x00"


def scope():
    network = os.getenv("QC_WALLET_LINK_NETWORK_ID", "")
    genesis = os.getenv("QC_WALLET_LINK_GENESIS_HEX", "").lower()
    if not re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}", network) or not re.fullmatch(r"[0-9a-f]{64}", genesis) or int(genesis, 16) == 0:
        raise ValueError("wallet_link_network_not_configured")
    return {"networkID": network, "genesisHex": genesis}


def schema(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS wallet_link_challenges_v1 (
        nonce TEXT PRIMARY KEY, player_id TEXT NOT NULL, address TEXT NOT NULL,
        challenge_json TEXT NOT NULL, expires_at INTEGER NOT NULL, consumed_at INTEGER)""")
    conn.execute("""CREATE TABLE IF NOT EXISTS wallet_link_proofs_v1 (
        address TEXT PRIMARY KEY, player_id TEXT UNIQUE NOT NULL, challenge_json TEXT NOT NULL,
        public_key TEXT NOT NULL, signature TEXT NOT NULL, verified_at INTEGER NOT NULL)""")
    conn.execute("""CREATE TABLE IF NOT EXISTS wallet_link_legacy_history_v1 (
        address TEXT NOT NULL, player_id TEXT NOT NULL, reclaimed_at INTEGER NOT NULL,
        PRIMARY KEY(address,player_id))""")


def issue(db, actor, address, now):
    try:
        owner = _wallet_owner(address)
        network = scope()
        if not re.fullmatch(r"tg_[0-9]{1,20}", actor or ""):
            raise ValueError("telegram_session_required")
        challenge = {"version":"QC_GAME_WALLET_LINK_V1", **network, "playerId":actor,
                     "address":address,"walletOwner":owner,"nonce":secrets.token_hex(32),
                     "issuedAt":now,"expiresAt":now+300}
        with closing(db()) as conn, conn:
            conn.execute("BEGIN IMMEDIATE")
            schema(conn)
            conn.execute("DELETE FROM wallet_link_challenges_v1 WHERE expires_at<=?", (now,))
            if conn.execute("SELECT COUNT(*) FROM wallet_link_challenges_v1").fetchone()[0] >= 4096:
                raise ValueError("wallet_challenge_capacity")
            conn.execute("INSERT INTO wallet_link_challenges_v1 VALUES(?,?,?,?,?,NULL)",
                         (challenge["nonce"],actor,address,canonical(challenge),now+300))
        return {"ok":True,"challenge":challenge,"signingBytesHex":(DOMAIN+canonical(challenge).encode("ascii")).hex()}
    except ValueError as error:
        return {"ok":False,"error":str(error)}


def consume(db, actor, address, nonce, public, signature, now):
    from cryptography.hazmat.primitives import hashes
    from cryptography.hazmat.primitives.asymmetric import ec
    from cryptography.exceptions import InvalidSignature
    try:
        network = scope()
        if not re.fullmatch(r"[0-9a-f]{64}", nonce or ""):
            raise ValueError("wallet_ownership_proof_required")
        with closing(db()) as conn, conn:
            conn.execute("BEGIN IMMEDIATE")
            schema(conn)
            row = conn.execute("SELECT challenge_json,consumed_at FROM wallet_link_challenges_v1 WHERE nonce=?", (nonce,)).fetchone()
            if not row or row[1] is not None:
                raise ValueError("wallet_challenge_missing_or_used")
            c = json.loads(row[0])
            if c["playerId"] != actor or c["address"] != address or any(c[k] != v for k,v in network.items()):
                raise ValueError("wallet_challenge_scope_mismatch")
            if not c["issuedAt"] <= now < c["expiresAt"]:
                raise ValueError("wallet_challenge_expired")
            if not re.fullmatch(r"04[0-9a-f]{128}", public or "") or not re.fullmatch(r"[0-9a-f]{16,144}",signature or ""):
                raise ValueError("invalid_wallet_signature")
            raw = bytes.fromhex(public)
            if hashlib.new("ripemd160",hashlib.sha256(raw[1:]).digest()).hexdigest() != c["walletOwner"]:
                raise ValueError("wallet_key_owner_mismatch")
            ec.EllipticCurvePublicKey.from_encoded_point(ec.SECP256K1(),raw).verify(
                bytes.fromhex(signature),DOMAIN+row[0].encode("ascii"),ec.ECDSA(hashes.SHA256()))
            proven = conn.execute("SELECT player_id FROM wallet_link_proofs_v1 WHERE address=?",(address,)).fetchone()
            if proven and proven[0] != actor:
                raise ValueError("wallet_already_verified_for_another_player")
            # Reclaim only unproven legacy bindings. Keep their history and all rewards.
            for (legacy,) in conn.execute("SELECT player_id FROM players WHERE qc_address=? AND player_id!=?",(address,actor)).fetchall():
                conn.execute("INSERT OR IGNORE INTO wallet_link_legacy_history_v1 VALUES(?,?,?)",(address,legacy,now))
                conn.execute("UPDATE players SET qc_address='' WHERE player_id=?",(legacy,))
            conn.execute("DELETE FROM wallet_link_proofs_v1 WHERE player_id=?",(actor,))
            conn.execute("INSERT INTO wallet_link_proofs_v1 VALUES(?,?,?,?,?,?)",(address,actor,row[0],public,signature,now))
            changed = conn.execute("UPDATE players SET qc_address=?,last_seen=? WHERE player_id=?",(address,now,actor)).rowcount
            if changed != 1:
                raise ValueError("player_record_required")
            conn.execute("UPDATE wallet_link_challenges_v1 SET consumed_at=? WHERE nonce=?",(now,nonce))
        return {"ok":True,"playerId":actor,"qcAddress":address,"ownershipVerified":True}
    except (InvalidSignature, WalletProofRejected):
        return {"ok":False,"error":"invalid_wallet_signature"}
    except ValueError as error:
        return {"ok":False,"error":str(error)}
