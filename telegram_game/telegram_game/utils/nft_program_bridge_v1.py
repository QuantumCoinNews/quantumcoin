"""Read the canonical node's verified game program; no local launch-date switch."""

import datetime
import ipaddress
import json
import os
import re
import urllib.request
from urllib.parse import urlsplit

VERSION = "QC_NFT_GAME_PROGRAM_HTTP_V1"
PROGRAM_PATH = "/api/nft/monthly/program"
MAX_RESPONSE_BYTES = 16384


def monthly_nft_periods_from_game_activation_v1(activation_at):
    try:
        ts = int(activation_at)
        if ts <= 0:
            return ()
        activated = datetime.datetime.fromtimestamp(ts, tz=datetime.timezone.utc)
        periods = []
        for offset in range(1, 13):
            index = activated.year * 12 + activated.month - 1 + offset
            year, month_zero = divmod(index, 12)
            if not 1 <= year <= 9999:
                return ()
            periods.append(f"{year:04d}-{month_zero + 1:02d}")
        return tuple(periods)
    except (TypeError, ValueError, OverflowError, OSError):
        return ()


def _validate_program_v1(data):
    if not isinstance(data, dict):
        return None
    if data.get("success") is not True or data.get("verified") is not True or data.get("version") != VERSION:
        return None
    activated = data.get("gameActivatedAt")
    if type(activated) is not int:
        return None
    periods = monthly_nft_periods_from_game_activation_v1(activated)
    if len(periods) != 12 or data.get("awardPeriods") != list(periods) or data.get("firstAwardPeriod") != periods[0]:
        return None
    if type(data.get("awardCount")) is not int or data["awardCount"] != 12:
        return None
    if type(data.get("maxAwardsPerMonth")) is not int or data["maxAwardsPerMonth"] != 1:
        return None
    for field in ("networkID", "campaignID"):
        value = data.get(field)
        if not isinstance(value, str) or not re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}", value):
            return None
    if not isinstance(data.get("legacyGenesisHex"), str) or not re.fullmatch(r"[0-9a-fA-F]{64}", data["legacyGenesisHex"]):
        return None
    if type(data.get("mintProtocolActive")) is not bool:
        return None
    return dict(data)


class _NoProgramRedirectV1(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def fetch_verified_nft_game_program_v1():
    # Resolve the same mode-aware origin used by the existing game connection.
    # Token is sent only in the canonical header; redirects never forward it.
    try:
        from telegram_game.testnet_connection_v1 import get_qc_api_base_for_mode_v1

        token = str(os.getenv("QC_API_TOKEN") or "").strip()
        if not token:
            return None
        base = get_qc_api_base_for_mode_v1().rstrip("/")
        parsed = urlsplit(base)
        if parsed.scheme not in {"http", "https"} or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
            return None
        if parsed.scheme == "http" and parsed.hostname.lower() != "localhost":
            if not ipaddress.ip_address(parsed.hostname).is_loopback:
                return None
        request = urllib.request.Request(base + PROGRAM_PATH, headers={"Accept": "application/json", "X-QC-API-KEY": token}, method="GET")
        opener = urllib.request.build_opener(_NoProgramRedirectV1())
        with opener.open(request, timeout=3.0) as response:
            if response.status != 200 or response.headers.get_content_type() != "application/json":
                return None
            body = response.read(MAX_RESPONSE_BYTES + 1)
        if len(body) > MAX_RESPONSE_BYTES:
            return None
        return _validate_program_v1(json.loads(body.decode("utf-8")))
    except Exception:
        # Connection/auth/format failure cannot become an activation date.
        # Do not include credentials, URLs, response bodies or exception text.
        return None
