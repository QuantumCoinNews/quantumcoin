# telegram_game/utils/qc_api.py
# cSpell:disable

from __future__ import annotations
# QC_URLOPEN_BALANCE_BRIDGE_V1 imports
import asyncio as _qc_asyncio
import json as _qc_json_stdlib
import time as _qc_time
import urllib.request as _qc_urllib_request



from typing import Any, Dict

import httpx

from telegram_game.config import get_api_base


async def fetch_health(timeout: float = 3.0) -> Dict[str, Any]:
    """
    QuantumCoin HTTP API'den /api/health bilgisi alır.
    API kapalı ya da hata verirse {"ok": False, "error": "..."} döner.
    """
    base = get_api_base().rstrip("/")
    url = f"{base}/api/health"

    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.get(url)
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

    if not addr:
        return {"ok": False, "error": "empty address"}

    url = f"{base}/api/wallet/balance/{addr}"

    def _read_json_sync() -> Dict[str, Any]:
        last_error = ""

        for attempt in range(1, 4):
            try:
                req = _qc_urllib_request.Request(
                    url,
                    headers={
                        "Accept": "application/json",
                        "Connection": "close",
                        "User-Agent": "QuantumCoin-TelegramGame/1.0",
                    },
                )

                with _qc_urllib_request.urlopen(req, timeout=float(timeout or 10.0)) as resp:
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
        async with httpx.AsyncClient(timeout=timeout) as client:
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
    """
    QuantumCoin node üzerinden gerçek blok kazdırır.
    Aktif QC node endpoint: POST /api/mine
    Beklenen cevap: success, height, reward, block_hash
    """
    base = get_api_base().rstrip("/")
    addr = (address or "").strip()

    if not addr:
        return {"ok": False, "success": False, "error": "empty address"}

    url = f"{base}/api/mine"
    payload = {"address": addr}

    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.post(url, json=payload)

        content_type = resp.headers.get("content-type", "")
        text_body = resp.text or ""

        resp.raise_for_status()

        if "application/json" not in content_type.lower():
            return {
                "ok": False,
                "success": False,
                "address": addr,
                "error": "non_json_response",
                "status_code": resp.status_code,
                "content_type": content_type,
                "preview": text_body[:160],
            }

        data = resp.json()

        if not isinstance(data, dict):
            return {
                "ok": False,
                "success": False,
                "address": addr,
                "error": "unexpected response",
            }

        data["ok"] = bool(data.get("success", data.get("ok", False)))
        data["address"] = addr
        return data

    except Exception as exc:
        return {
            "ok": False,
            "success": False,
            "address": addr,
            "error": str(exc),
        }


# QC_NODE_SEND_TX_BRIDGE_V1
async def fetch_send_tx(
    from_address: str,
    to_address: str,
    amount: str,
    priv_hex: str,
    timeout: float = 30.0,
) -> Dict[str, Any]:
    """
    QuantumCoin node üzerinden gerçek QC transaction gönderir.
    Aktif QC node endpoint: POST /api/tx/send
    Beklenen payload: from, to, amount, priv_hex
    """
    base = get_api_base().rstrip("/")

    from_addr = (from_address or "").strip()
    to_addr = (to_address or "").strip()
    amount_str = str(amount or "").strip()
    priv = (priv_hex or "").strip()

    if not from_addr:
        return {"ok": False, "success": False, "error": "empty from address"}
    if not to_addr:
        return {"ok": False, "success": False, "error": "empty to address"}
    if not amount_str:
        return {"ok": False, "success": False, "error": "empty amount"}
    if not priv:
        return {"ok": False, "success": False, "error": "empty priv_hex"}

    url = f"{base}/api/tx/send"
    payload = {
        "from": from_addr,
        "to": to_addr,
        "amount": amount_str,
        "priv_hex": priv,
    }

    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.post(url, json=payload)

        content_type = resp.headers.get("content-type", "")
        text_body = resp.text or ""

        resp.raise_for_status()

        if "application/json" not in content_type.lower():
            return {
                "ok": False,
                "success": False,
                "from": from_addr,
                "to": to_addr,
                "amount": amount_str,
                "error": "non_json_response",
                "status_code": resp.status_code,
                "content_type": content_type,
                "preview": text_body[:160],
            }

        data = resp.json()

        if not isinstance(data, dict):
            return {
                "ok": False,
                "success": False,
                "from": from_addr,
                "to": to_addr,
                "amount": amount_str,
                "error": "unexpected response",
            }

        data["ok"] = bool(data.get("success", data.get("ok", False)))
        data["from"] = from_addr
        data["to"] = to_addr
        data["amount"] = amount_str

        return data

    except Exception as exc:
        return {
            "ok": False,
            "success": False,
            "from": from_addr,
            "to": to_addr,
            "amount": amount_str,
            "error": str(exc),
        }
