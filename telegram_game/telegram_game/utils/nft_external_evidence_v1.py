"""Network adapter: fixed local origins, HTTPS, bounded reads, no redirects.

Witness POST contains public commitment only. QC/node credentials are never
forwarded. Lost publication response recovers the ONE fixed public scope URL;
no alternative witness, network or beacon round is selected.
"""
import json
import os
import sqlite3
import time
import urllib.error
import urllib.request
from urllib.parse import urlsplit

from . import nft_draw_evidence_v1 as draw
from .nft_publication_witness_v1 import publication_id_v1
from .nft_wallet_proof_v1 import canonical, WalletProofRejected
from .nft_selection_identity_v1 import SelectionRejected


def https_url(value, origin_only=False):
    draw.require(isinstance(value, str) and value.isascii() and 0 < len(value) <= 2048
                 and "\\" not in value and all(ord(c) > 32 for c in value), "invalid_external_service_url")
    u = urlsplit(value)
    draw.require(u.scheme == "https" and u.hostname and not (u.username or u.password or u.query or u.fragment)
                 and (not origin_only or u.path in {"", "/"}), "invalid_external_service_url")
    return value.rstrip("/")


def transport_policy(policy):
    services = policy.get("externalServices")
    draw.require(isinstance(services, dict) and set(services) == {"witnessSubmitURL", "beaconBaseURL"}, "external_services_not_configured")
    submit = https_url(services["witnessSubmitURL"])
    beacon = https_url(services["beaconBaseURL"], True)
    prefix = policy["publicationWitness"]["urlPrefix"]
    https_url(prefix)
    draw.require(urlsplit(submit).netloc == urlsplit(prefix).netloc, "witness_origin_mismatch")
    return submit, beacon


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class ExternalEvidenceClientV1:
    def __init__(self, policy):
        self.policy = policy
        self.submit_url, self.beacon_base = transport_policy(policy)
        self.opener = urllib.request.build_opener(_NoRedirect())

    def call(self, url, body=None):
        headers = {"Accept": "application/json"}
        if body is not None:
            token = os.getenv("QC_NFT_WITNESS_SUBMIT_TOKEN", "")
            draw.require(token and token.isascii() and not any(ord(c) < 33 or ord(c) > 126 for c in token), "witness_submit_credentials_required")
            headers.update({"X-QMA-Witness-Token": token, "Content-Type": "application/json"})
        request = urllib.request.Request(url, data=canonical(body).encode("ascii") if body is not None else None, headers=headers)
        try:
            with self.opener.open(request, timeout=10) as response:
                draw.require(response.status in {200, 201} and response.headers.get_content_type() == "application/json", "external_evidence_unavailable")
                raw = response.read(16385)
                draw.require(len(raw) <= 16384, "external_evidence_too_large")
            from .nft_mint_bridge_v1 import _unique_object
            return json.loads(raw, object_pairs_hook=_unique_object,
                              parse_constant=lambda _: (_ for _ in ()).throw(ValueError()))
        except urllib.error.HTTPError as error:
            error.close()
            raise draw.DrawRejected("external_evidence_unavailable") from None
        except (OSError, ValueError):
            raise draw.DrawRejected("external_evidence_unavailable") from None

    def publish(self, commitment, frozen_at):
        url = self.policy["publicationWitness"]["urlPrefix"] + publication_id_v1(commitment)
        try:
            receipt = self.call(self.submit_url, {"commitment": commitment})
            draw.verify_publication_v1(commitment, receipt, self.policy, frozen_at)
            draw.require(receipt["statement"]["entryURL"] == url, "publication_location_mismatch")
        except draw.DrawRejected:
            # The external namespace is first-write-only. Recover the same
            # signed record even when the POST succeeded but its reply was lost.
            pass
        receipt = self.call(url)
        draw.verify_publication_v1(commitment, receipt, self.policy, frozen_at)
        draw.require(receipt["statement"]["entryURL"] == url, "publication_location_mismatch")
        return receipt

    def beacon(self, commitment):
        chain = commitment["beaconChainHash"]
        info = self.call(self.beacon_base + "/" + chain + "/info")
        p = self.policy["beacon"]
        draw.require(isinstance(info, dict) and info.get("hash") == chain
                     and info.get("public_key") == p["publicKey"] and info.get("schemeID") == p["scheme"]
                     and type(info.get("period")) is int and info["period"] == p["periodSeconds"]
                     and type(info.get("genesis_time")) is int and info["genesis_time"] == p["genesisTime"], "beacon_info_pin_mismatch")
        raw = self.call(self.beacon_base + "/" + chain + "/public/" + str(commitment["round"]))
        draw.require(isinstance(raw, dict) and set(raw) == {"round", "signature", "randomness"}, "invalid_beacon_evidence")
        result = {"chainHash": chain, **raw}
        draw.verify_beacon_v1(commitment, result)
        return result


def publish_and_register_v1(connect, program, policy, period, now=lambda: int(time.time())):
    conn = None
    try:
        client = ExternalEvidenceClientV1(policy)
        conn = connect()
        conn.execute("BEGIN IMMEDIATE")
        frozen = draw.pool(conn, period)
        draw.require_pool_wallets(conn, frozen, program)
        commitment = draw.commitment_v1(frozen, program, policy)
        draw.require(frozen["frozenAt"] <= now() < commitment["beaconAt"], "commitment_registration_too_late")
        conn.execute("CREATE TABLE IF NOT EXISTS monthly_nft_publication_outbox_v1(period TEXT PRIMARY KEY,commitment_json TEXT NOT NULL)")
        prior = conn.execute("SELECT commitment_json FROM monthly_nft_publication_outbox_v1 WHERE period=?", (period,)).fetchone()
        raw = canonical(commitment)
        draw.require(prior is None or prior[0] == raw, "publication_intent_conflict")
        if prior is None:
            conn.execute("INSERT INTO monthly_nft_publication_outbox_v1 VALUES(?,?)", (period, raw))
        conn.commit()
        conn.close(); conn = None
        receipt = client.publish(commitment, frozen["frozenAt"])
        return draw.register_draw_commitment_v1(connect, program, policy, period, receipt, now())
    except (draw.DrawRejected, WalletProofRejected, SelectionRejected) as error:
        return {"ok": False, "error": str(error)}
    except (OSError, sqlite3.Error, ValueError, KeyError, TypeError):
        return {"ok": False, "error": "external_evidence_unavailable"}
    finally:
        if conn is not None:
            conn.close()


def draw_from_beacon_v1(connect, program, policy, period, serialize, now=lambda: int(time.time())):
    conn = None
    try:
        conn = connect()
        commitment = draw.commitment_v1(draw.pool(conn, period), program, policy)
        draw.require(now() >= commitment["beaconAt"], "beacon_round_not_released")
        # Missing publication is rejected BEFORE making any beacon request.
        row = conn.execute("SELECT commitment_json FROM monthly_nft_draw_commitments_v1 WHERE period=?", (period,)).fetchone()
        draw.require(row is not None and row[0] == canonical(commitment), "draw_commitment_required")
        conn.close(); conn = None
        beacon = ExternalEvidenceClientV1(policy).beacon(commitment)
        return draw.draw_monthly_winner_v1(connect, program, policy, period, beacon, now(), serialize)
    except (draw.DrawRejected, WalletProofRejected, SelectionRejected) as error:
        return {"ok": False, "error": str(error)}
    except (OSError, sqlite3.Error, ValueError, KeyError, TypeError):
        return {"ok": False, "error": "external_evidence_unavailable"}
    finally:
        if conn is not None:
            conn.close()
