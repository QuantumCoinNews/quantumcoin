# telegram_game/utils/qc_api.py
# cSpell:disable

from __future__ import annotations

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
async def fetch_address_balance(address: str, timeout: float = 3.0) -> Dict[str, Any]:
    """
    QuantumCoin node üzerinden gerçek QC adres bakiyesini okur.
    Aktif QC node endpoint: /api/wallet/balance/{addr}
    """
    base = get_api_base().rstrip("/")
    addr = (address or "").strip()

    if not addr:
        return {"ok": False, "error": "empty address"}

    url = f"{base}/api/wallet/balance/{addr}"

    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.get(url)

        content_type = resp.headers.get("content-type", "")
        text_body = resp.text or ""

        resp.raise_for_status()

        if "application/json" not in content_type.lower():
            return {
                "ok": False,
                "address": addr,
                "error": "non_json_response",
                "status_code": resp.status_code,
                "content_type": content_type,
                "preview": text_body[:160],
            }

        data = resp.json()

        if not isinstance(data, dict):
            return {"ok": False, "address": addr, "error": "unexpected response"}

        if "ok" not in data:
            data["ok"] = True

        data["address"] = addr
        return data

    except Exception as exc:
        return {
            "ok": False,
            "address": addr,
            "error": str(exc),
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
