"""Deploy on an INDEPENDENT operator/host, not inside the game/node trust root.

Public immutable GET records; authenticated, first-commitment-only publication
per network/campaign/genesis/period. The operator supplies its Ed25519 key,
clock, persistent DB and HTTPS public origin. Never creates production keys.
"""
import hashlib
import hmac
import json
import sqlite3

from .nft_draw_evidence_v1 import (COMMITMENT_VERSION, WITNESS_VERSION, BEACON_SIZES,
                                  hex_bytes, DrawRejected, require)
from .nft_wallet_proof_v1 import canonical

SCOPE_FIELDS = ("networkID", "campaignID", "legacyGenesisHex")


def publication_id_v1(commitment):
    scope = {k: commitment[k] for k in (*SCOPE_FIELDS, "period")}
    return hashlib.sha256(b"QC_NFT_PUBLICATION_SCOPE_V1\x00" + canonical(scope).encode("ascii")).hexdigest()


def create_witness_app_v1(connect, signer, base_url, submit_token, allowed_scope, now):
    from fastapi import FastAPI, Request
    from fastapi.responses import JSONResponse
    from urllib.parse import urlsplit
    from .nft_mint_bridge_v1 import _unique_object, _identifier
    from .nft_draw_freeze_v1 import _period_end
    from .nft_draw_evidence_v1 import digest
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

    require(isinstance(signer, Ed25519PrivateKey), "witness_signer_required")
    parsed = urlsplit(base_url)
    require(parsed.scheme == "https" and parsed.hostname and not (parsed.username or parsed.password or parsed.query or parsed.fragment)
            and parsed.path in {"", "/"}, "witness_https_origin_required")
    require(isinstance(submit_token, str) and 32 <= len(submit_token) <= 256
            and all(33 <= ord(c) <= 126 for c in submit_token), "witness_submit_token_required")
    require(isinstance(allowed_scope, dict) and set(allowed_scope) == set(SCOPE_FIELDS), "witness_program_scope_required")
    for k in ("networkID", "campaignID"):
        _identifier(allowed_scope[k])
    hex_bytes(allowed_scope["legacyGenesisHex"], 32)
    base_url = base_url.rstrip("/")
    app = FastAPI(title="Independent monthly NFT publication witness", docs_url=None, redoc_url=None)

    def tables(conn):
        conn.execute("""CREATE TABLE IF NOT EXISTS nft_publication_records_v1 (
            entry_id TEXT PRIMARY KEY, commitment_json TEXT NOT NULL,
            receipt_json TEXT NOT NULL)""")
        for op in ("UPDATE", "DELETE"):
            conn.execute(f"CREATE TRIGGER IF NOT EXISTS nft_publication_records_v1_{op.lower()} BEFORE {op} ON nft_publication_records_v1 BEGIN SELECT RAISE(ABORT,'publication_immutable'); END")

    def response(status, code):
        return JSONResponse(status_code=status, content={"ok": False, "error": code})

    @app.post("/v1/commitments")
    async def publish(request: Request):
        supplied = request.headers.get("X-QMA-Witness-Token", "")
        if not supplied.isascii() or not hmac.compare_digest(supplied, submit_token):
            return response(401, "witness_authentication_required")
        conn = None
        try:
            body = bytearray()
            async for chunk in request.stream():
                body.extend(chunk)
                if len(body) > 16384:
                    return response(413, "witness_request_too_large")
            value = json.loads(bytes(body), object_pairs_hook=_unique_object,
                               parse_constant=lambda _: (_ for _ in ()).throw(ValueError()))
            require(isinstance(value, dict) and set(value) == {"commitment"}, "invalid_witness_request")
            c = value["commitment"]
            require(isinstance(c, dict) and set(c) == {"version", *SCOPE_FIELDS, "period", "poolHash", "candidateCount", "policyHash", "beaconChainHash", "beaconScheme", "beaconPublicKey", "round", "beaconAt"}
                    and c["version"] == COMMITMENT_VERSION, "invalid_witness_commitment")
            require(all(c[k] == v for k, v in allowed_scope.items()), "witness_program_scope_mismatch")
            _period_end(c["period"])
            for field in ("poolHash", "policyHash", "beaconChainHash"):
                hex_bytes(c[field], 32)
            require(isinstance(c["beaconScheme"], str) and c["beaconScheme"] in BEACON_SIZES, "unsupported_beacon_scheme")
            hex_bytes(c["beaconPublicKey"], BEACON_SIZES[c["beaconScheme"]][0])
            require(type(c["candidateCount"]) is int and 1 <= c["candidateCount"] <= 10000
                    and type(c["round"]) is int and 1 <= c["round"] < 2**64
                    and type(c["beaconAt"]) is int, "invalid_witness_commitment")
            entry = publication_id_v1(c)
            raw = canonical(c)
            conn = connect()
            conn.execute("BEGIN IMMEDIATE")
            tables(conn)
            prior = conn.execute("SELECT commitment_json,receipt_json FROM nft_publication_records_v1 WHERE entry_id=?", (entry,)).fetchone()
            if prior:
                if prior[0] != raw:
                    return response(409, "witness_period_already_committed")
                return JSONResponse(content=json.loads(prior[1]))
            published = now()
            require(type(published) is int and 0 < published < c["beaconAt"], "witness_beacon_deadline_passed")
            statement = {"version": WITNESS_VERSION, "commitmentHash": digest(COMMITMENT_VERSION, c),
                         "publishedAt": published, "entryURL": base_url + "/entries/" + entry}
            signature = signer.sign(WITNESS_VERSION.encode("ascii") + b"\x00" + canonical(statement).encode("ascii")).hex()
            receipt = {"statement": statement, "signature": signature}
            conn.execute("INSERT INTO nft_publication_records_v1 VALUES(?,?,?)", (entry, raw, canonical(receipt)))
            conn.commit()
            return JSONResponse(status_code=201, content=receipt)
        except (ValueError, KeyError, TypeError, OverflowError):
            return response(422, "invalid_witness_commitment")
        except sqlite3.Error:
            return response(503, "witness_storage_unavailable")
        finally:
            if conn is not None:
                conn.close()

    @app.get("/entries/{entry_id}")
    def read_entry(entry_id: str):
        conn = None
        try:
            hex_bytes(entry_id, 32)
            conn = connect()
            exists = conn.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='nft_publication_records_v1'").fetchone()
            if not exists:
                return response(404, "witness_record_not_found")
            row = conn.execute("SELECT receipt_json FROM nft_publication_records_v1 WHERE entry_id=?", (entry_id,)).fetchone()
            return JSONResponse(content=json.loads(row[0])) if row else response(404, "witness_record_not_found")
        except (ValueError, TypeError):
            return response(404, "witness_record_not_found")
        except sqlite3.Error:
            return response(503, "witness_storage_unavailable")
        finally:
            if conn is not None:
                conn.close()

    return app
