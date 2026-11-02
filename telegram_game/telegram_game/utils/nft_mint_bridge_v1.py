"""Controlled monthly NFT transport. Node-local authorities remain authoritative.

Only public, signed award evidence crosses HTTP. A pending SQLite intent is not
an award. Confirmation requires the exact record on the node's canonical chain
and configured chain file. This is a snapshot confirmation, not finality.
"""

import datetime
import hashlib
import ipaddress
import json
import os
import re
import sqlite3
import struct
import time
import urllib.error
import urllib.request
from pathlib import Path
from urllib.parse import urlsplit

from .nft_program_bridge_v1 import _validate_program_v1

REQUEST_VERSION = "QC_NFT_MONTHLY_MINT_REQUEST_V1"
RECEIPT_VERSION = "QC_NFT_MONTHLY_MINT_RECEIPT_V1"
BRIDGE_VERSION = "QC_TG_NFT_MINT_BRIDGE_V1"
MAX_BYTES = 65536


class NFTMintBridgeError(ValueError):
    """Codes are fixed strings; credentials and remote bodies are never echoed."""


def _require(condition, code):
    if not condition:
        raise NFTMintBridgeError(code)


def _hex(value, size):
    _require(isinstance(value, str) and re.fullmatch(r"[0-9a-fA-F]{%d}" % (size * 2), value), "invalid_award_hex")
    return bytes.fromhex(value)


def _identifier(value):
    _require(isinstance(value, str) and re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}", value), "invalid_award_scope")
    return value


def canonical_award_v1(payload):
    """Match blockchain/nft_record_v1.go byte for byte; never sign here."""
    _require(isinstance(payload, dict) and set(payload) == {"version", "record", "decision", "approvals"}, "invalid_award_schema")
    _require(payload["version"] == REQUEST_VERSION, "invalid_award_version")
    r, d, approvals = payload["record"], payload["decision"], payload["approvals"]
    _require(isinstance(r, dict) and set(r) == {"networkID", "campaignID", "awardPeriod", "serial", "recipientPubKeyHash", "metadataHash"}, "invalid_award_record")
    _require(isinstance(d, dict) and set(d) == {"networkID", "legacyGenesisHex", "recordDigest"}, "invalid_award_decision")
    network, campaign = _identifier(r["networkID"]), _identifier(r["campaignID"])
    period = r["awardPeriod"]
    _require(isinstance(period, str) and re.fullmatch(r"[0-9]{4}-[0-9]{2}", period), "invalid_award_period")
    try:
        _require(datetime.datetime.strptime(period, "%Y-%m").strftime("%Y-%m") == period, "invalid_award_period")
    except ValueError:
        raise NFTMintBridgeError("invalid_award_period") from None
    _require(type(r["serial"]) is int and 1 <= r["serial"] <= 12, "invalid_award_serial")
    owner, metadata = _hex(r["recipientPubKeyHash"], 20), _hex(r["metadataHash"], 32)
    _require(any(metadata), "metadata_commitment_required")
    identity = b"QC_NFT_TOKEN_ID_V1\x00"
    for value in (network, campaign, period):
        raw = value.encode("ascii")
        identity += struct.pack(">H", len(raw)) + raw
    identity += struct.pack(">I", r["serial"]) + metadata
    token = hashlib.sha256(identity).hexdigest()
    digest = hashlib.sha256(b"QC_NFT_MINT_RECORD_V1\x00" + identity + owner).hexdigest()
    _require(d["networkID"] == network and _hex(d["recordDigest"], 32).hex() == digest, "award_digest_mismatch")
    genesis = _hex(d["legacyGenesisHex"], 32).hex()
    _require(isinstance(approvals, list) and 1 <= len(approvals) <= 32, "award_approvals_required")
    seen = set()
    for a in approvals:
        _require(isinstance(a, dict) and set(a) == {"publicKey", "signature"}, "invalid_award_approval")
        public = _hex(a["publicKey"], 65)
        signature = a["signature"]
        _require(public[0] == 4 and public not in seen, "invalid_award_signer")
        _require(isinstance(signature, str) and 8 <= len(signature) <= 132 and len(signature) % 2 == 0, "invalid_award_signature")
        _hex(signature, len(signature) // 2)
        seen.add(public)
    # Detach caller-owned structures before any I/O or persistence.
    raw = json.dumps(payload, separators=(",", ":"), allow_nan=False).encode()
    _require(len(raw) <= MAX_BYTES, "award_request_too_large")
    return json.loads(raw), token, digest, genesis


def validate_receipt_v1(payload, receipt):
    _, token, digest, genesis = canonical_award_v1(payload)
    _require(isinstance(receipt, dict), "invalid_mint_receipt")
    _require(receipt.get("success") is True and receipt.get("persisted") is True and receipt.get("version") == RECEIPT_VERSION, "unconfirmed_mint_receipt")
    _require(type(receipt.get("alreadyMinted")) is bool, "invalid_mint_receipt")
    _require(_hex(receipt.get("tokenID"), 32).hex() == token and _hex(receipt.get("recordDigest"), 32).hex() == digest, "receipt_commitment_mismatch")
    _require(_hex(receipt.get("legacyGenesisHex"), 32).hex() == genesis, "receipt_genesis_mismatch")
    for field in ("networkID", "campaignID", "awardPeriod", "serial"):
        _require(type(receipt.get(field)) is type(payload["record"][field]) and receipt[field] == payload["record"][field], "receipt_scope_mismatch")
    for field, size in (("recipientPubKeyHash", 20), ("metadataHash", 32)):
        _require(_hex(receipt.get(field), size) == _hex(payload["record"][field], size), "receipt_recipient_or_metadata_mismatch")
    for field in ("mintHeight", "chainHeight", "persistedHeight"):
        _require(type(receipt.get(field)) is int and receipt[field] >= 1, "invalid_receipt_height")
    _require(receipt["mintHeight"] <= min(receipt["chainHeight"], receipt["persistedHeight"]), "invalid_receipt_height")
    for field in ("mintBlockHash", "chainTipHash", "persistedTipHash"):
        _hex(receipt.get(field), 32)
    for height, tip in (("chainHeight", "chainTipHash"), ("persistedHeight", "persistedTipHash")):
        if receipt[height] == receipt["mintHeight"]:
            _require(receipt[tip].lower() == receipt["mintBlockHash"].lower(), "receipt_tip_mismatch")
    # Keep only the defined public receipt schema, never arbitrary remote data.
    fields = {"success", "version", "persisted", "alreadyMinted", "tokenID", "recordDigest", "legacyGenesisHex", "mintHeight", "chainHeight", "persistedHeight", "mintBlockHash", "chainTipHash", "persistedTipHash"} | set(payload["record"])
    _require(set(receipt) == fields, "invalid_receipt_schema")
    return json.loads(json.dumps(receipt))


def _unique_object(pairs):
    result = {}
    for key, value in pairs:
        _require(key not in result, "ambiguous_node_response")
        result[key] = value
    return result


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class NodeNFTMintClientV1:
    def __init__(self):
        from telegram_game.testnet_connection_v1 import get_qc_api_base_for_mode_v1
        self.base = get_qc_api_base_for_mode_v1().rstrip("/")
        self.token = str(os.getenv("QC_API_TOKEN") or "").strip()
        parsed = urlsplit(self.base)
        _require(self.token and parsed.scheme in {"http", "https"} and parsed.hostname and not (parsed.username or parsed.password or parsed.query or parsed.fragment or parsed.path), "node_connection_not_configured")
        if parsed.scheme == "http":
            try:
                local = parsed.hostname.lower() == "localhost" or ipaddress.ip_address(parsed.hostname).is_loopback
            except ValueError:
                local = False
            _require(local, "node_https_required")
        self.opener = urllib.request.build_opener(_NoRedirect())

    def _call(self, path, payload=None):
        data = None if payload is None else json.dumps(payload, separators=(",", ":"), allow_nan=False).encode()
        request = urllib.request.Request(self.base + path, data=data, method="GET" if payload is None else "POST", headers={"Accept": "application/json", "Content-Type": "application/json", "X-QC-API-KEY": self.token})
        try:
            with self.opener.open(request, timeout=10.0) as response:
                status = response.status
                if response.headers.get_content_type() != "application/json":
                    return 0, None
                body = response.read(MAX_BYTES + 1)
            if len(body) > MAX_BYTES:
                return 0, None
            result = json.loads(body.decode("utf-8"), object_pairs_hook=_unique_object, parse_constant=lambda _: (_ for _ in ()).throw(ValueError()))
            return status, result
        except urllib.error.HTTPError as error:
            status = error.code
            error.close()
            return status, None
        except Exception:
            return 0, None

    def verified_program(self):
        status, result = self._call("/api/nft/monthly/program")
        program = _validate_program_v1(result) if status == 200 else None
        _require(program is not None and program["mintProtocolActive"] is True, "verified_program_unavailable")
        return program

    def confirm_or_mint(self, payload, allow_mint):
        payload, _, _, _ = canonical_award_v1(payload)
        status, receipt = self._call("/api/nft/monthly/receipt", payload)
        if status == 200:
            return validate_receipt_v1(payload, receipt)
        if status == 404 and not allow_mint:
            raise NFTMintBridgeError("canonical_mint_receipt_missing")
        if status != 404 or not allow_mint:
            raise NFTMintBridgeError("mint_confirmation_pending")
        # Exactly one mint POST per attempt; uncertainty is reconciled by READ.
        status, receipt = self._call("/api/nft/monthly/mint", payload)
        if status == 200:
            try:
                return validate_receipt_v1(payload, receipt)
            except NFTMintBridgeError:
                pass
        status, receipt = self._call("/api/nft/monthly/receipt", payload)
        if status == 200:
            return validate_receipt_v1(payload, receipt)
        raise NFTMintBridgeError("mint_confirmation_pending")


def _wallet_owner(address):
    alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
    _require(isinstance(address, str) and 1 <= len(address) <= 128, "linked_wallet_invalid")
    number = 0
    for char in address:
        _require(char in alphabet, "linked_wallet_invalid")
        number = number * 58 + alphabet.index(char)
    raw = (b"\x00" * (len(address) - len(address.lstrip("1")))) + number.to_bytes((number.bit_length() + 7) // 8, "big")
    _require(len(raw) == 25 and raw[0] == 0 and hashlib.sha256(hashlib.sha256(raw[:21]).digest()).digest()[:4] == raw[21:], "linked_wallet_invalid")
    return raw[1:21].hex()


def _file_hash(root, relative, maximum):
    _require(isinstance(relative, str) and relative, "catalog_file_required")
    path = (root / relative).resolve()
    _require(path.is_relative_to(root.resolve()) and path.is_file(), "catalog_file_unavailable")
    with path.open("rb") as stream:
        data = stream.read(maximum + 1)
    _require(0 < len(data) <= maximum, "catalog_file_invalid")
    return hashlib.sha256(data).hexdigest()


def _catalog_metadata(root, catalog_id, period, program):
    path = root / "nft/catalog/monthly_rare_nft_stock_2026.json"
    _require(path.is_file() and path.stat().st_size <= MAX_BYTES, "catalog_unavailable")
    stock = json.loads(path.read_text(encoding="utf-8-sig"), object_pairs_hook=_unique_object)
    _require(isinstance(stock, dict), "catalog_not_ready")
    items = stock.get("items", [])
    _require(isinstance(items, list) and len(items) == 12 and all(isinstance(item, dict) for item in items), "catalog_not_ready")
    _require(type(stock.get("totalPlanned")) is int and stock["totalPlanned"] == 12 and type(stock.get("awardsPerMonth")) is int and stock["awardsPerMonth"] == 1, "catalog_not_ready")
    _require({item.get("period") for item in items} == set(program["awardPeriods"]), "catalog_program_periods_mismatch")
    for field in ("id", "name", "concept", "imagePath", "metadataPath", "finalImageSha256", "finalMetadataSha256"):
        _require(len({item.get(field) for item in items}) == 12 and all(item.get(field) for item in items), "catalog_not_unique_or_ready")
    for field in ("finalImageSha256", "finalMetadataSha256"):
        _require(len({_hex(item.get(field), 32).hex() for item in items}) == 12, "catalog_hashes_not_unique")
    chosen = None
    for item in items:
        _require(item.get("finalReady") is True and item.get("rarity") == "rare" and item.get("artworkStatus") == "FINAL_READY" and item.get("metadataStatus") == "FINAL_READY" and item.get("uniquenessCheck") == "PASSED" and item.get("aiReviewStatus") == "APPROVED" and item.get("adminApprovalStatus") == "APPROVED", "catalog_not_ready")
        image_hash, metadata_hash = _hex(item.get("finalImageSha256"), 32).hex(), _hex(item.get("finalMetadataSha256"), 32).hex()
        _require(_file_hash(root, item.get("imagePath"), 16 * 1024 * 1024) == image_hash and _file_hash(root, item.get("metadataPath"), 1024 * 1024) == metadata_hash, "catalog_hash_mismatch")
        if item["id"] == catalog_id and item["period"] == period:
            chosen = metadata_hash
    _require(chosen is not None, "catalog_item_period_mismatch")
    return chosen


def _candidate(conn, candidate_id):
    row = conn.execute("SELECT user_id, period, ai_review_status, ai_risk_level, admin_status, winner_status, minted_status FROM monthly_nft_candidates WHERE candidate_id=?", (candidate_id,)).fetchone()
    _require(row is not None, "candidate_not_found")
    player = conn.execute("SELECT qc_address FROM players WHERE player_id=?", (row[0],)).fetchone()
    _require(player is not None, "linked_wallet_required")
    address = player[0]
    _require(conn.execute("SELECT COUNT(*) FROM players WHERE qc_address=?", (address,)).fetchone()[0] == 1, "linked_wallet_not_unique")
    return tuple(row), address, _wallet_owner(address)


def _ledger(conn):
    conn.execute("""CREATE TABLE IF NOT EXISTS monthly_nft_mint_intents_v1 (
        candidate_id TEXT PRIMARY KEY, network_id TEXT NOT NULL, campaign_id TEXT NOT NULL,
        award_period TEXT NOT NULL, record_digest TEXT NOT NULL, binding_json TEXT NOT NULL,
        request_json TEXT NOT NULL, receipt_json TEXT, status TEXT NOT NULL,
        updated_at INTEGER NOT NULL, UNIQUE(network_id,award_period))""")


def _lifecycle_evidence(conn, root, program, candidate_id, catalog_id, period, owner, metadata):
    from .nft_draw_evidence_v1 import load_policy_v1, require_draw_receipt_v1
    from .nft_final_readiness_v1 import require_final_approval_v1
    policy = load_policy_v1(program)
    now = int(time.time())
    draw_hash = require_draw_receipt_v1(conn, program, policy, candidate_id, period, owner, now)
    approval_hash = require_final_approval_v1(root, program, policy, candidate_id, catalog_id,
                                             period, owner, metadata, draw_hash, now)
    return [draw_hash, approval_hash]


def controlled_monthly_nft_mint_v1(connect, repository_root, candidate_id, catalog_id, signed_award):
    """Invoked only by the existing authenticated admin route; defaults closed."""
    if os.getenv("QC_ENABLE_CHAIN_NFT_MINT", "").strip() != "1" or os.getenv("QC_ENABLE_TG_NFT_MINT_BRIDGE", "").strip() != "1":
        return {"ok": False, "error": "telegram_nft_mint_bridge_disabled", "mintConfirmed": False}
    try:
        from telegram_game.testnet_connection_v1 import telegram_game_chain_mode_v1
        _require(telegram_game_chain_mode_v1() == "mainnet", "monthly_nft_mainnet_mode_required")
        _require(isinstance(candidate_id, str) and 0 < len(candidate_id) <= 128 and isinstance(catalog_id, str) and 0 < len(catalog_id) <= 128, "candidate_and_catalog_required")
        payload, token, digest, genesis = canonical_award_v1(signed_award)
        client = NodeNFTMintClientV1()
        program = client.verified_program()
        record = payload["record"]
        _require(record["networkID"] == program["networkID"] and record["campaignID"] == program["campaignID"] and genesis == program["legacyGenesisHex"].lower(), "award_program_mismatch")
        _require(record["awardPeriod"] in program["awardPeriods"] and program["awardPeriods"].index(record["awardPeriod"]) + 1 == record["serial"], "award_program_period_mismatch")
        root = Path(repository_root).resolve()
        metadata = _catalog_metadata(root, catalog_id, record["awardPeriod"], program)
        _require(metadata == record["metadataHash"].lower(), "award_catalog_hash_mismatch")
        conn = connect()
        try:
            conn.execute("BEGIN IMMEDIATE")
            row, address, owner = _candidate(conn, candidate_id)
            _require(row[1] == record["awardPeriod"] and owner == record["recipientPubKeyHash"].lower(), "award_candidate_recipient_mismatch")
            evidence = _lifecycle_evidence(conn, root, program, candidate_id, catalog_id, row[1], owner, metadata)
            _ledger(conn)
            existing = conn.execute("SELECT record_digest,binding_json,status FROM monthly_nft_mint_intents_v1 WHERE candidate_id=?", (candidate_id,)).fetchone()
            binding = json.dumps([row[0], row[1], address, catalog_id, genesis, *evidence], separators=(",", ":"))
            _require(existing is None or (existing[0] == digest and existing[1] == binding), "candidate_mint_intent_conflict")
            confirmed = existing is not None and existing[2] in {"CONFIRMED", "RECONCILIATION_REQUIRED"}
            if not confirmed:
                _require(row[2:6] == ("AI_REVIEW_PASSED", "LOW", "APPROVED_BY_ADMIN", "SELECTED_MONTHLY_WINNER") and row[6] == "NOT_MINTED", "candidate_not_approved_for_mint")
                start = datetime.datetime.strptime(row[1], "%Y-%m").replace(tzinfo=datetime.timezone.utc)
                end = start.replace(year=start.year + 1, month=1) if start.month == 12 else start.replace(month=start.month + 1)
                sessions = conn.execute("SELECT COUNT(*) FROM monthly_nft_verified_game_sessions WHERE user_id=? AND activity_verified=1 AND completed_at>=? AND completed_at<?", (row[0], int(start.timestamp()), int(end.timestamp()))).fetchone()[0]
                _require(sessions > 0, "verified_game_session_required")
            _require(row[6] in {"NOT_MINTED", "MINTED_CONFIRMED", "MINT_RECONCILIATION_REQUIRED"}
                     and (row[6] == "NOT_MINTED" or confirmed), "candidate_mint_state_conflict")
            if existing is None:
                conn.execute("INSERT INTO monthly_nft_mint_intents_v1 VALUES(?,?,?,?,?,?,?,?,?,?)", (candidate_id, record["networkID"], record["campaignID"], row[1], digest, binding, json.dumps(payload, separators=(",", ":")), None, "PENDING", int(time.time())))
            conn.commit()
        finally:
            conn.close()
        try:
            receipt = client.confirm_or_mint(payload, allow_mint=not confirmed)
        except NFTMintBridgeError as error:
            if str(error) == "canonical_mint_receipt_missing" and confirmed:
                # A previously confirmed mint disappeared from the canonical
                # node. Preserve the intent and recipient; never re-mint or
                # silently leave the game displaying a confirmed NFT.
                conn = connect()
                try:
                    conn.execute("BEGIN IMMEDIATE")
                    actual = conn.execute("SELECT record_digest,binding_json FROM monthly_nft_mint_intents_v1 WHERE candidate_id=?", (candidate_id,)).fetchone()
                    _require(actual == (digest, binding), "candidate_mint_intent_conflict")
                    conn.execute("UPDATE monthly_nft_mint_intents_v1 SET status='RECONCILIATION_REQUIRED',updated_at=? WHERE candidate_id=?", (int(time.time()), candidate_id))
                    conn.execute("UPDATE monthly_nft_candidates SET minted_status='MINT_RECONCILIATION_REQUIRED',updated_at=? WHERE candidate_id=?", (int(time.time()), candidate_id))
                    conn.commit()
                finally:
                    conn.close()
            raise
        # The intent is committed before network I/O; receipt + candidate are
        # committed together. A DB failure can be retried through receipt read.
        conn = connect()
        try:
            conn.execute("BEGIN IMMEDIATE")
            current, current_address, current_owner = _candidate(conn, candidate_id)
            _require(current[0:2] == row[0:2] and current_address == address and current_owner == owner, "candidate_binding_changed")
            _require(_catalog_metadata(root, catalog_id, row[1], program) == metadata, "catalog_changed_during_mint")
            _require(_lifecycle_evidence(conn, root, program, candidate_id, catalog_id, row[1], owner, metadata) == evidence, "evidence_changed_during_mint")
            actual = conn.execute("SELECT record_digest,binding_json FROM monthly_nft_mint_intents_v1 WHERE candidate_id=?", (candidate_id,)).fetchone()
            _require(actual is not None and actual[0] == digest and actual[1] == binding, "candidate_mint_intent_conflict")
            conn.execute("UPDATE monthly_nft_mint_intents_v1 SET status='CONFIRMED',receipt_json=?,updated_at=? WHERE candidate_id=?", (json.dumps(receipt, separators=(",", ":")), int(time.time()), candidate_id))
            conn.execute("UPDATE monthly_nft_candidates SET minted_status='MINTED_CONFIRMED',updated_at=? WHERE candidate_id=?", (int(time.time()), candidate_id))
            conn.commit()
        finally:
            conn.close()
        return {"ok": True, "version": BRIDGE_VERSION, "candidateId": candidate_id, "mintConfirmed": True, "receipt": receipt}
    except NFTMintBridgeError as error:
        return {"ok": False, "version": BRIDGE_VERSION, "error": str(error), "mintConfirmed": False}
    except (OSError, sqlite3.Error, ValueError, TypeError, KeyError, OverflowError):
        return {"ok": False, "version": BRIDGE_VERSION, "error": "mint_bridge_confirmation_unavailable", "mintConfirmed": False}
