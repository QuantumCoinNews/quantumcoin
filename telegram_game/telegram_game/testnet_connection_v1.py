# QC_TG_TESTNET_CONNECTION_V1
# Separates Telegram Game local/main and testnet QuantumCoin API connections.
# This module does not mine, transfer, mint, settle, select winners, or change balances.

import os
from urllib.parse import urlparse

QC_TG_TESTNET_CONNECTION_VERSION = "QC_TG_TESTNET_CONNECTION_V1"

TG_CHAIN_MODE_LOCAL = "local"
TG_CHAIN_MODE_TESTNET = "testnet"
TG_CHAIN_MODE_MAINNET = "mainnet"

DEFAULT_LOCAL_QC_API_BASE = "http://127.0.0.1:8082"
DEFAULT_TESTNET_QC_API_BASE = "http://127.0.0.1:18082"
NETWORK_LABEL_PATH = "/api/network/label"


class TelegramGameConnectionError(ValueError):
    pass


def _env(name: str, default: str = "") -> str:
    return str(os.getenv(name) or default).strip()


def _env_bool(name: str) -> bool:
    return _env(name).lower() in {"1", "true", "yes", "on"}


def _contains_testnet(value: str) -> bool:
    return "testnet" in str(value or "").strip().lower()


def telegram_game_chain_mode_v1() -> str:
    raw = (
        _env("TELEGRAM_GAME_CHAIN_MODE")
        or _env("TELEGRAM_GAME_NETWORK")
        or _env("QC_CHAIN_MODE")
        or _env("QC_NETWORK")
        or _env("QC_MODE")
    ).lower()

    if _env_bool("QC_TESTNET") or _env_bool("QC_PUBLIC_TESTNET"):
        return TG_CHAIN_MODE_TESTNET

    if _contains_testnet(raw) or _contains_testnet(_env("QC_NET_MAGIC")):
        return TG_CHAIN_MODE_TESTNET

    if raw in {"mainnet", "main", "production", "prod"}:
        return TG_CHAIN_MODE_MAINNET

    return TG_CHAIN_MODE_LOCAL


def _normalize_base(base: str) -> str:
    return str(base or "").strip().rstrip("/")


def get_qc_api_base_for_mode_v1() -> str:
    mode = telegram_game_chain_mode_v1()
    configured = _normalize_base(
        _env("QC_API_BASE")
        or _env("QUANTUMCOIN_API_BASE")
        or _env("QC_NODE_API")
    )

    if configured:
        validate_qc_api_base_for_mode_v1(configured, mode)
        return configured

    if mode == TG_CHAIN_MODE_TESTNET:
        return DEFAULT_TESTNET_QC_API_BASE

    return DEFAULT_LOCAL_QC_API_BASE


def validate_qc_api_base_for_mode_v1(base: str, mode: str = None) -> None:
    mode = (mode or telegram_game_chain_mode_v1()).lower()
    base = _normalize_base(base)

    if not base:
        raise TelegramGameConnectionError("QC API base is required")

    parsed = urlparse(base)
    port = parsed.port
    host = (parsed.hostname or "").lower()

    if mode == TG_CHAIN_MODE_TESTNET:
        if port in {8081, 8082} and not _env_bool("TELEGRAM_GAME_ALLOW_TESTNET_MAIN_PORT"):
            raise TelegramGameConnectionError(
                "Telegram Game testnet mode must not use local/main QC API port 8081/8082"
            )
        return

    if port == 18082 and mode != TG_CHAIN_MODE_TESTNET:
        raise TelegramGameConnectionError(
            "Telegram Game testnet QC API port 18082 requires explicit testnet mode"
        )

    if "testnet" in host and mode != TG_CHAIN_MODE_TESTNET:
        raise TelegramGameConnectionError(
            "Telegram Game testnet host requires explicit testnet mode"
        )


def telegram_game_connection_info_v1() -> dict:
    mode = telegram_game_chain_mode_v1()
    base = get_qc_api_base_for_mode_v1()

    return {
        "version": QC_TG_TESTNET_CONNECTION_VERSION,
        "telegramGameChainMode": mode,
        "qcApiBase": base,
        "networkLabelPath": NETWORK_LABEL_PATH,
        "isTestnet": mode == TG_CHAIN_MODE_TESTNET,
        "testnetSimulatedMiningOnly": mode == TG_CHAIN_MODE_TESTNET,
        "realChainMiningAllowed": mode != TG_CHAIN_MODE_TESTNET,
        "safeBoundary": {
            "doesNotMine": True,
            "doesNotTransfer": True,
            "doesNotMint": True,
            "doesNotSettle": True,
            "doesNotChangeBalances": True,
            "doesNotChangeConsensus": True,
            "doesNotChangeRewards": True,
        },
    }


def assert_telegram_game_connection_safe_v1() -> dict:
    info = telegram_game_connection_info_v1()
    validate_qc_api_base_for_mode_v1(info["qcApiBase"], info["telegramGameChainMode"])
    return info
