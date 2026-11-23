# telegram_game/utils/qc_api.py
# cSpell:disable

from __future__ import annotations
# QC_URLOPEN_BALANCE_BRIDGE_V1 imports
import asyncio as _qc_asyncio
import json as _qc_json_stdlib
import time as _qc_time
import urllib.request as _qc_urllib_request
import urllib.error as _qc_urllib_error



from typing import Any, Dict

import httpx
import os
from urllib.parse import quote

from telegram_game.config import get_api_base


class _NoNodeRedirect(_qc_urllib_request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


async def fetch_runtime_policy(timeout=5.0):
    """Only the configured authenticated node supplies runtime policy facts."""
    try:
        async with httpx.AsyncClient(timeout=timeout, headers={"X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")}, follow_redirects=False, trust_env=False) as client:
            response = await client.get(get_api_base() + "/api/runtime/policy")
        response.raise_for_status()
        data = response.json()
        if not isinstance(data, dict) or data.get("version") != "NODE_RUNTIME_POLICY_V1" or data.get("source") != "quantumcoin_go_node" or not isinstance(data.get("context"), dict) or not isinstance(data.get("sections"), dict):
            raise ValueError("invalid_node_policy")
        return data
    except Exception:
        # No response body, origin credentials, token or Python default fallback.
        return {"ok": False, "error": "node_runtime_policy_unavailable"}


def node_urlopen(request, timeout):
    try:
        return _qc_urllib_request.build_opener(_qc_urllib_request.ProxyHandler({}), _NoNodeRedirect()).open(request, timeout=timeout)
    except _qc_urllib_error.HTTPError as error:
        # Redirects and HTTP failures are rejected, never consumed as receipts.
        # HTTPError owns the response stream even when urlopen raised it.
        error.close()
        raise


async def fetch_health(timeout: float = 3.0) -> Dict[str, Any]:
    """
    QuantumCoin HTTP API'den /api/health bilgisi alır.
    API kapalı ya da hata verirse {"ok": False, "error": "..."} döner.
    """
    base = get_api_base().rstrip("/")
    url = f"{base}/api/health"

    try:
        async with httpx.AsyncClient(timeout=timeout, headers={"X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")}, follow_redirects=False, trust_env=False) as client:
            resp = await client.get(url, headers={"X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")})
        if resp.status_code == 401 and 'QuantumCoin API' in resp.headers.get('www-authenticate', ''):
            return {"ok": False, "reachable": True, "authenticated": False, "error": "node_authentication_required"}
        resp.raise_for_status()
        data = resp.json()
        if not isinstance(data, dict):
            return {"ok": False, "error": "unexpected response"}
        # Sağlıklı cevapta ok alanı yoksa biz ekleyelim
        if "ok" not in data:
            data["ok"] = True
        return data
    except Exception as exc:
        return {"ok": False, "error": str(exc)}


# QC_NODE_READ_BRIDGE_V1
async def fetch_address_balance(address: str, timeout: float = 10.0) -> Dict[str, Any]:
    """
    QuantumCoin node üzerinden gerçek QC adres bakiyesini okur.

    QC_URLOPEN_BALANCE_BRIDGE_V1:
    Local Windows runtime'da httpx bazı /api/wallet/balance çağrılarında ReadTimeout'a düşebiliyor.
    Aynı endpoint PowerShell/urllib ile JSON döndüğü için bu helper urllib + asyncio.to_thread kullanır.
    """
    base = get_api_base().rstrip("/")
    addr = (address or "").strip()

    if not addr or len(addr) > 128:
        return {"ok": False, "error": "empty address"}

    url = f"{base}/api/wallet/balance/{quote(addr, safe='')}"

    def _read_json_sync() -> Dict[str, Any]:
        last_error = ""

        for attempt in range(1, 4):
            try:
                req = _qc_urllib_request.Request(
                    url,
                    headers={
                        "Accept": "application/json",
                        "X-QC-API-KEY": os.getenv("QC_API_TOKEN", ""),
                        "Connection": "close",
                        "User-Agent": "QuantumCoin-TelegramGame/1.0",
                    },
                )

                with node_urlopen(req, timeout=float(timeout or 10.0)) as resp:
                    status_code = int(getattr(resp, "status", 200) or 200)
                    content_type = resp.headers.get("Content-Type", "")
                    text_body = resp.read().decode("utf-8", errors="replace")

                if status_code < 200 or status_code >= 300:
                    return {
                        "ok": False,
                        "address": addr,
                        "error": "http_status",
                        "status_code": status_code,
                        "content_type": content_type,
                        "preview": text_body[:160],
                    }

                if "application/json" not in str(content_type).lower():
                    return {
                        "ok": False,
                        "address": addr,
                        "error": "non_json_response",
                        "status_code": status_code,
                        "content_type": content_type,
                        "preview": text_body[:160],
                    }

                try:
                    data = _qc_json_stdlib.loads(text_body)
                except Exception as exc:
                    return {
                        "ok": False,
                        "address": addr,
                        "error": "invalid_json",
                        "errorType": exc.__class__.__name__,
                        "preview": text_body[:160],
                    }

                if not isinstance(data, dict):
                    return {"ok": False, "address": addr, "error": "unexpected response"}

                if "ok" not in data:
                    data["ok"] = True

                data["address"] = addr
                return data

            except Exception as exc:
                last_error = f"{exc.__class__.__name__}: {str(exc)}"
                _qc_time.sleep(0.25 * attempt)

        return {
            "ok": False,
            "address": addr,
            "error": last_error,
            "errorType": "urllib_retry_failed",
        }

    try:
        return await _qc_asyncio.to_thread(_read_json_sync)
    except Exception as exc:
        return {
            "ok": False,
            "address": addr,
            "error": str(exc),
            "errorType": exc.__class__.__name__,
        }


async def fetch_tx_status(txid: str, timeout: float = 3.0) -> Dict[str, Any]:
    """
    QuantumCoin node üzerinden tx durumunu okur.
    """
    base = get_api_base().rstrip("/")
    txid = (txid or "").strip()

    if not txid:
        return {"ok": False, "error": "empty txid"}

    url = f"{base}/api/tx/status"

    try:
        async with httpx.AsyncClient(timeout=timeout, headers={"X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")}, follow_redirects=False, trust_env=False) as client:
            resp = await client.get(url, params={"id": txid})
        resp.raise_for_status()
        data = resp.json()

        if not isinstance(data, dict):
            return {"ok": False, "error": "unexpected response"}

        if "ok" not in data:
            data["ok"] = True

        data["txid"] = txid
        return data

    except Exception as exc:
        return {
            "ok": False,
            "txid": txid,
            "error": str(exc),
        }


# QC_NODE_MINE_BRIDGE_V1
async def fetch_mine_block(address: str, timeout: float = 180.0) -> Dict[str, Any]:
    """Retired: only submit client-solved canonical work through game_pow."""
    return {"ok": False, "success": False, "error": "server_side_game_mining_retired"}


# QC_NODE_SEND_TX_BRIDGE_V1
async def fetch_send_tx(
    from_address: str,
    to_address: str,
    amount: str,
    priv_hex: str,
    timeout: float = 30.0,
) -> Dict[str, Any]:
    """Retired: signing belongs to the wallet, never an HTTP bridge."""
    return {"ok": False, "success": False, "error": "wallet_local_signing_required"}

async def fetch_submit_signed_tx(transaction: dict, timeout: float = 30.0) -> Dict[str, Any]:
    def has_float(value):
        if isinstance(value, float):
            return True
        if isinstance(value, dict):
            return any(has_float(item) for item in value.values())
        return isinstance(value, list) and any(has_float(item) for item in value)
    if has_float(transaction):
        return {"ok": False, "success": False, "error": "integer_protocol_amount_required"}
    base = get_api_base().rstrip("/")
    if not isinstance(transaction, dict) or not set(transaction).issubset({"ID", "Inputs", "Outputs", "Timestamp", "Sender", "Amount"}):
        return {"ok": False, "success": False, "error": "signed_transaction_required"}
    if not isinstance(transaction.get("Inputs"), list) or not isinstance(transaction.get("Outputs"), list):
        return {"ok": False, "success": False, "error": "signed_transaction_required"}
    for item in transaction["Inputs"]:
        if not isinstance(item, dict) or not set(item).issubset({"TxID", "OutIndex", "Signature", "PubKey"}):
            return {"ok": False, "success": False, "error": "signed_transaction_required"}
    for item in transaction["Outputs"]:
        if not isinstance(item, dict) or not set(item).issubset({"Amount", "PubKeyHash"}):
            return {"ok": False, "success": False, "error": "signed_transaction_required"}
        if type(item.get("Amount")) is not int or not 0 < item["Amount"] <= (1 << 63) - 1:
            return {"ok": False, "success": False, "error": "integer_protocol_amount_required"}
    if len(_qc_json_stdlib.dumps(transaction)) > 1048576:
        return {"ok": False, "success": False, "error": "signed_transaction_size_limit"}
    try:
        async with httpx.AsyncClient(timeout=timeout, follow_redirects=False, trust_env=False) as client:
            response = await client.post(base + "/api/tx/send", json={"transaction": transaction}, headers={"X-QC-API-KEY": os.getenv("QC_API_TOKEN", "")})
        if response.status_code != 200:
            return {"ok": False, "success": False, "error": "signed_transaction_rejected"}
        data = response.json()
        if not isinstance(data, dict):
            return {"ok": False, "success": False, "error": "invalid_node_response"}
        data["ok"] = bool(data.get("success"))
        return data
    except Exception:
        return {"ok": False, "success": False, "error": "node_request_failed"}
