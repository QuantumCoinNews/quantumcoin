"""Verified monthly draw: pinned local trust, external pre-beacon publication,
real drand BLS signature, wallet control, unbiased selection, atomic receipt.

Supported: pedersen-bls-unchained and quicknet's bls-unchained-g1-rfc9380.
Other schemes fail closed; scheme/public-key/signature sizes cannot be swapped.
Protocol: https://github.com/drand/drand/blob/master/crypto/schemes.go
The external witness must attest PUBLIC publication before the chosen round;
a local DB timestamp or administrator-signed self-attestation is insufficient.
No witness/beacon/approval keys are supplied by this module or by HTTP callers.
"""
import hashlib
import json
import os
import re
import sqlite3
from pathlib import Path
from urllib.parse import urlsplit

from .nft_draw_freeze_v1 import _get_existing, _COLUMNS
from .nft_selection_identity_v1 import SelectionRejected, _identity
from .nft_wallet_proof_v1 import (canonical, program_scope, require_wallet_proof_v1,
                                  WalletProofRejected)

VERSION = "QC_NFT_VERIFIED_DRAW_V1"
COMMITMENT_VERSION = "QC_NFT_DRAW_COMMITMENT_V1"
WITNESS_VERSION = "QC_NFT_PUBLICATION_WITNESS_V1"
BEACON_SIZES = {"pedersen-bls-unchained": (48, 96),
                "bls-unchained-g1-rfc9380": (96, 48)}


class DrawRejected(ValueError):
    pass


def require(ok, code):
    if not ok:
        raise DrawRejected(code)


def digest(domain, value):
    return hashlib.sha256(domain.encode("ascii") + b"\x00" + canonical(value).encode("ascii")).hexdigest()


def hex_bytes(value, length):
    require(isinstance(value, str) and re.fullmatch(r"[0-9a-f]{%d}" % (length * 2), value), "invalid_evidence_hex")
    return bytes.fromhex(value)


def validate_policy(policy, program):
    scope = program_scope(program)
    require(isinstance(policy, dict) and policy.get("version") == "QC_NFT_EVIDENCE_POLICY_V1", "evidence_policy_required")
    require(all(policy.get(k) == v for k, v in scope.items()), "evidence_policy_program_mismatch")
    beacon = policy.get("beacon")
    require(isinstance(beacon, dict) and set(beacon) == {"chainHash", "publicKey", "scheme", "genesisTime", "periodSeconds", "commitLeadSeconds"}, "invalid_beacon_policy")
    require(isinstance(beacon["scheme"], str) and beacon["scheme"] in BEACON_SIZES, "unsupported_beacon_scheme")
    hex_bytes(beacon["chainHash"], 32)
    hex_bytes(beacon["publicKey"], BEACON_SIZES[beacon["scheme"]][0])
    for name in ("genesisTime", "periodSeconds", "commitLeadSeconds"):
        require(type(beacon[name]) is int and beacon[name] > 0, "invalid_beacon_policy")
    require(beacon["commitLeadSeconds"] >= 60, "beacon_commit_lead_too_short")
    witness = policy.get("publicationWitness")
    require(isinstance(witness, dict) and set(witness) == {"publicKey", "urlPrefix"}, "invalid_witness_policy")
    hex_bytes(witness["publicKey"], 32)
    url = urlsplit(witness["urlPrefix"])
    require(url.scheme == "https" and bool(url.hostname) and not (url.username or url.password or url.query or url.fragment)
            and witness["urlPrefix"].endswith("/"), "invalid_witness_policy")
    return policy


def load_policy_v1(program):
    # Local deployment configuration only. No request-provided policy/key URL.
    name = os.getenv("QC_NFT_EVIDENCE_POLICY_PATH", "")
    require(bool(name), "evidence_policy_not_configured")
    path = Path(name)
    require(path.is_absolute() and path.is_file() and path.stat().st_size <= 16384, "evidence_policy_unavailable")
    from .nft_mint_bridge_v1 import _unique_object
    policy = json.loads(path.read_text(encoding="utf-8-sig"), object_pairs_hook=_unique_object)
    return validate_policy(policy, program)


def commitment_v1(frozen, program, policy):
    validate_policy(policy, program)
    manifest = frozen["manifest"]
    require(manifest["period"] in program["awardPeriods"], "draw_period_outside_program")
    b = policy["beacon"]
    # A deterministic future round, bound to the immutable freeze and policy.
    round_number = max(1, (manifest["frozenAt"] + b["commitLeadSeconds"] - b["genesisTime"]) // b["periodSeconds"] + 2)
    require(round_number < 2**64, "invalid_beacon_round")
    return {"version": COMMITMENT_VERSION, **program_scope(program),
            "period": manifest["period"], "poolHash": frozen["poolHash"],
            "candidateCount": frozen["candidateCount"], "policyHash": digest("QC_NFT_EVIDENCE_POLICY_V1", policy),
            "beaconChainHash": b["chainHash"], "beaconScheme": b["scheme"],
            "beaconPublicKey": b["publicKey"], "round": round_number,
            "beaconAt": b["genesisTime"] + (round_number - 1) * b["periodSeconds"]}


def verify_ed25519(public_hex, signature_hex, domain, statement):
    try:
        from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
        from cryptography.exceptions import InvalidSignature
    except ImportError:
        raise DrawRejected("evidence_crypto_dependency_unavailable") from None
    public, signature = hex_bytes(public_hex, 32), hex_bytes(signature_hex, 64)
    try:
        Ed25519PublicKey.from_public_bytes(public).verify(signature, domain.encode("ascii") + b"\x00" + canonical(statement).encode("ascii"))
    except (ValueError, InvalidSignature):
        raise DrawRejected("invalid_external_evidence_signature") from None


def verify_publication_v1(commitment, publication, policy, frozen_at):
    require(isinstance(publication, dict) and set(publication) == {"statement", "signature"}, "publication_evidence_required")
    s = publication["statement"]
    require(isinstance(s, dict) and set(s) == {"version", "commitmentHash", "publishedAt", "entryURL"}, "invalid_publication_statement")
    require(s["version"] == WITNESS_VERSION and s["commitmentHash"] == digest(COMMITMENT_VERSION, commitment), "publication_commitment_mismatch")
    require(type(s["publishedAt"]) is int and frozen_at <= s["publishedAt"] < commitment["beaconAt"], "publication_not_before_beacon")
    require(isinstance(s["entryURL"], str) and len(s["entryURL"]) <= 2048
            and s["entryURL"].startswith(policy["publicationWitness"]["urlPrefix"])
            and not urlsplit(s["entryURL"]).fragment, "publication_location_mismatch")
    verify_ed25519(policy["publicationWitness"]["publicKey"], publication["signature"], WITNESS_VERSION, s)


def verify_beacon_v1(commitment, beacon):
    require(isinstance(beacon, dict) and set(beacon) == {"chainHash", "round", "signature", "randomness"}, "invalid_beacon_evidence")
    require(type(beacon["round"]) is int and beacon["round"] == commitment["round"]
            and beacon["chainHash"] == commitment["beaconChainHash"], "beacon_scope_mismatch")
    scheme = commitment.get("beaconScheme", "pedersen-bls-unchained")
    require(isinstance(scheme, str) and scheme in BEACON_SIZES, "unsupported_beacon_scheme")
    signature = hex_bytes(beacon["signature"], BEACON_SIZES[scheme][1])
    random = hashlib.sha256(signature).hexdigest()
    require(beacon["randomness"] == random, "beacon_randomness_mismatch")
    try:
        from py_ecc.bls import G2Basic
    except ImportError:
        raise DrawRejected("beacon_crypto_dependency_unavailable") from None
    message = hashlib.sha256(beacon["round"].to_bytes(8, "big")).digest()
    if scheme == "pedersen-bls-unchained":
        valid = G2Basic.Verify(hex_bytes(commitment["beaconPublicKey"], 48), message, signature)
    else:
        # Use py_ecc's point decoding, subgroup checks, RFC9380 hash-to-curve
        # and pairing primitives. No custom curve/arithmetic implementation.
        from py_ecc.bls.g2_primitives import signature_to_G2, pubkey_to_G1, subgroup_check
        from py_ecc.bls.hash_to_curve import hash_to_G1
        from py_ecc.optimized_bls12_381 import G2, is_inf, pairing
        try:
            key = signature_to_G2(hex_bytes(commitment["beaconPublicKey"], 96))
            sig = pubkey_to_G1(signature)
            require(not is_inf(key) and not is_inf(sig) and subgroup_check(key) and subgroup_check(sig), "invalid_beacon_signature")
            hashed = hash_to_G1(message, b"BLS_SIG_BLS12381G1_XMD:SHA-256_SSWU_RO_NUL_", hashlib.sha256)
            valid = pairing(key, hashed) == pairing(G2, sig)
        except (ValueError, AssertionError):
            valid = False
    require(valid, "invalid_beacon_signature")
    return bytes.fromhex(random)


def unbiased_index_v1(commitment_hash, randomness, count):
    require(type(count) is int and 0 < count <= 10000, "invalid_draw_candidate_count")
    require(isinstance(randomness, bytes) and len(randomness) == 32, "invalid_draw_randomness")
    seed = b"QC_NFT_DRAW_REJECTION_V1\x00" + hex_bytes(commitment_hash, 32) + randomness
    limit = 2**256 - (2**256 % count)
    for counter in range(10000):
        value = int.from_bytes(hashlib.sha256(seed + counter.to_bytes(4, "big")).digest(), "big")
        if value < limit:
            return value % count, counter
    raise DrawRejected("draw_rejection_limit")


def tables(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_draw_commitments_v1 (
        period TEXT PRIMARY KEY, commitment_json TEXT NOT NULL,
        publication_json TEXT NOT NULL, registered_at INTEGER NOT NULL)""")
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_draw_receipts_v1 (
        period TEXT PRIMARY KEY, receipt_json TEXT NOT NULL, receipt_hash TEXT NOT NULL)""")
    for name in ("monthly_nft_draw_commitments_v1", "monthly_nft_draw_receipts_v1"):
        for operation in ("UPDATE", "DELETE"):
            conn.execute(f"CREATE TRIGGER IF NOT EXISTS {name}_{operation.lower()} BEFORE {operation} ON {name} BEGIN SELECT RAISE(ABORT,'draw_evidence_immutable'); END")


def pool(conn, period):
    exists = conn.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='monthly_nft_pool_freezes_v1'").fetchone()
    require(exists, "monthly_pool_not_frozen")
    frozen = _get_existing(conn, period)
    require(frozen is not None, "monthly_pool_not_frozen")
    return frozen


def require_pool_wallets(conn, frozen, program):
    proofs = {}
    for c in frozen["manifest"]["candidates"]:
        proofs[c["candidateId"]] = require_wallet_proof_v1(conn, program, c["userId"], frozen["period"], c["walletOwner"], frozen["frozenAt"])
    return proofs


def register_draw_commitment_v1(connect, program, policy, period, publication, now):
    conn = None
    try:
        require(type(now) is int, "invalid_draw_time")
        conn = connect()
        conn.execute("BEGIN IMMEDIATE")
        frozen = pool(conn, period)
        commitment = commitment_v1(frozen, program, policy)
        verify_publication_v1(commitment, publication, policy, frozen["frozenAt"])
        require(publication["statement"]["publishedAt"] <= now < commitment["beaconAt"], "commitment_registration_too_late")
        require_pool_wallets(conn, frozen, program)
        tables(conn)
        raw, pub = canonical(commitment), canonical(publication)
        prior = conn.execute("SELECT commitment_json,publication_json FROM monthly_nft_draw_commitments_v1 WHERE period=?", (period,)).fetchone()
        require(prior is None or prior == (raw, pub), "draw_commitment_conflict")
        if prior is None:
            conn.execute("INSERT INTO monthly_nft_draw_commitments_v1 VALUES(?,?,?,?)", (period, raw, pub, now))
        conn.commit()
        return {"ok": True, "commitment": commitment, "alreadyRegistered": prior is not None}
    except (DrawRejected, SelectionRejected, WalletProofRejected) as error:
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, KeyError, TypeError, OSError):
        return {"ok": False, "error": "draw_evidence_unavailable"}
    finally:
        if conn is not None:
            conn.close()


def receipt_from_evidence(conn, program, policy, period, beacon, now):
    frozen = pool(conn, period)
    commitment = commitment_v1(frozen, program, policy)
    row = conn.execute("SELECT commitment_json,publication_json,registered_at FROM monthly_nft_draw_commitments_v1 WHERE period=?", (period,)).fetchone()
    require(row is not None and row[0] == canonical(commitment), "draw_commitment_required")
    publication = json.loads(row[1])
    verify_publication_v1(commitment, publication, policy, frozen["frozenAt"])
    require(publication["statement"]["publishedAt"] <= row[2] < commitment["beaconAt"] <= now, "invalid_draw_time")
    proofs = require_pool_wallets(conn, frozen, program)
    random = verify_beacon_v1(commitment, beacon)
    index, rejected = unbiased_index_v1(digest(COMMITMENT_VERSION, commitment), random, frozen["candidateCount"])
    return {"version": VERSION, "commitment": commitment, "publication": publication,
            "beacon": beacon, "winner": frozen["manifest"]["candidates"][index],
            "winnerIndex": index, "rejectionCount": rejected, "walletProofs": proofs}


def draw_monthly_winner_v1(connect, program, policy, period, beacon, now, serialize):
    conn = None
    try:
        require(type(now) is int, "invalid_draw_time")
        conn = connect()
        conn.execute("BEGIN IMMEDIATE")
        validate_policy(policy, program)
        tables(conn)
        receipt = receipt_from_evidence(conn, program, policy, period, beacon, now)
        raw, hashed = canonical(receipt), digest(VERSION, receipt)
        prior = conn.execute("SELECT receipt_json,receipt_hash FROM monthly_nft_draw_receipts_v1 WHERE period=?", (period,)).fetchone()
        if prior:
            require(prior == (raw, hashed), "draw_receipt_conflict")
            winner_state = conn.execute("SELECT user_id,period,winner_status FROM monthly_nft_candidates WHERE candidate_id=?", (receipt["winner"]["candidateId"],)).fetchone()
            require(winner_state == (receipt["winner"]["userId"], period, "SELECTED_MONTHLY_WINNER"), "draw_winner_state_conflict")
        else:
            # Check all frozen identities again, and reject replaced/legacy state.
            for frozen_candidate in pool(conn, period)["manifest"]["candidates"]:
                row = conn.execute("SELECT " + _COLUMNS + " FROM monthly_nft_candidates WHERE candidate_id=?", (frozen_candidate["candidateId"],)).fetchone()
                require(row is not None, "draw_candidate_missing")
                c = serialize(row)
                require(c["userId"] == frozen_candidate["userId"] and c["period"] == period, "draw_candidate_binding_changed")
                require(c["nftType"] == "rare" and c["aiReviewStatus"] == "AI_REVIEW_PASSED" and c["aiRiskLevel"] == "LOW"
                        and c["adminStatus"] == "APPROVED_BY_ADMIN" and c["mintedStatus"] == "NOT_MINTED"
                        and c["winnerStatus"] != "SELECTED_MONTHLY_WINNER", "draw_candidate_not_approved")
                _identity(conn, c, now)
            require(conn.execute("SELECT 1 FROM monthly_nft_candidates WHERE period=? AND (winner_status='SELECTED_MONTHLY_WINNER' OR minted_status IS NOT 'NOT_MINTED')", (period,)).fetchone() is None, "monthly_winner_already_selected")
            conn.execute("CREATE UNIQUE INDEX IF NOT EXISTS idx_monthly_nft_one_winner_identity_v1 ON monthly_nft_candidates(period) WHERE winner_status='SELECTED_MONTHLY_WINNER' OR minted_status IS NOT 'NOT_MINTED'")
            conn.execute("INSERT INTO monthly_nft_draw_receipts_v1 VALUES(?,?,?)", (period, raw, hashed))
            conn.execute("UPDATE monthly_nft_candidates SET winner_status='SELECTED_MONTHLY_WINNER',updated_at=? WHERE candidate_id=?",
                         (now, receipt["winner"]["candidateId"]))
        conn.commit()
        return {"ok": True, "receipt": receipt, "receiptHash": hashed, "alreadySelected": prior is not None}
    except (DrawRejected, SelectionRejected, WalletProofRejected) as error:
        return {"ok": False, "error": str(error)}
    except (sqlite3.Error, ValueError, KeyError, TypeError, OSError):
        return {"ok": False, "error": "draw_evidence_unavailable"}
    finally:
        if conn is not None:
            conn.close()


def require_draw_receipt_v1(conn, program, policy, candidate_id, period, owner, now):
    validate_policy(policy, program)
    row = conn.execute("SELECT receipt_json,receipt_hash FROM monthly_nft_draw_receipts_v1 WHERE period=?", (period,)).fetchone()
    require(row is not None, "verified_draw_receipt_required")
    receipt = json.loads(row[0])
    expected = receipt_from_evidence(conn, program, policy, period, receipt["beacon"], now)
    require(row == (canonical(expected), digest(VERSION, expected)), "draw_receipt_tampered")
    require(receipt["winner"]["candidateId"] == candidate_id and receipt["winner"]["walletOwner"] == owner, "draw_award_recipient_mismatch")
    return row[1]
