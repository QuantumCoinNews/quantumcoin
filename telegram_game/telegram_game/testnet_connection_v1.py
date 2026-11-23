# QC_TG_TESTNET_CONNECTION_V1
# Separates Telegram Game local/main and testnet QuantumCoin API connections.
# This module does not mine, transfer, mint, settle, select winners, or change balances.

import os
from ipaddress import ip_address
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
    value = _env(name).lower()
    if value in {"1", "true", "yes", "on"}:
        return True
    if value not in {"", "0", "false", "no", "off"}:
        raise TelegramGameConnectionError(f"{name} requires a boolean value")
    return False


def telegram_game_chain_mode_v1() -> str:
    aliases = {"local": TG_CHAIN_MODE_LOCAL, "local-release": TG_CHAIN_MODE_LOCAL,
               "testnet": TG_CHAIN_MODE_TESTNET, "testnet-local": TG_CHAIN_MODE_TESTNET,
               "testnet-public": TG_CHAIN_MODE_TESTNET, "mainnet": TG_CHAIN_MODE_MAINNET}
    game_aliases = {**aliases, "main": TG_CHAIN_MODE_MAINNET, "production": TG_CHAIN_MODE_MAINNET,
                    "prod": TG_CHAIN_MODE_MAINNET}
    modes = set()
    for key in ("TELEGRAM_GAME_CHAIN_MODE", "TELEGRAM_GAME_NETWORK", "QC_NETWORK",
                "QC_NETWORK_MODE", "QC_CHAIN_MODE", "QC_MODE"):
        value = _env(key).lower()
        if not value:
            continue
        selected = (game_aliases if key.startswith("TELEGRAM_GAME_") else aliases).get(value)
        if selected is None:
            raise TelegramGameConnectionError(f"{key} contains an unknown network profile")
        modes.add(selected)
    for key in ("QC_TESTNET", "QC_PUBLIC_TESTNET", "QC_API_PUBLIC_TESTNET", "QC_TESTNET_PUBLIC_API"):
        if _env_bool(key):
            modes.add(TG_CHAIN_MODE_TESTNET)
    if _env_bool("QC_MAINNET"):
        modes.add(TG_CHAIN_MODE_MAINNET)
    magic = _env("QC_NET_MAGIC").upper()
    if magic:
        network = {"QC_TESTNET_V1": TG_CHAIN_MODE_TESTNET, "QC_MAINNET_V1": TG_CHAIN_MODE_MAINNET,
                   "QC_LOCAL_RELEASE_V1": TG_CHAIN_MODE_LOCAL}.get(magic)
        if network is None:
            raise TelegramGameConnectionError("QC_NET_MAGIC contains an unknown network identity")
        modes.add(network)
    if len(modes) > 1:
        raise TelegramGameConnectionError("conflicting QuantumCoin network settings")
    return next(iter(modes), TG_CHAIN_MODE_LOCAL)


def _normalize_base(base: str) -> str:
    return str(base or "").strip().rstrip("/")


def get_qc_api_base_for_mode_v1() -> str:
    mode = telegram_game_chain_mode_v1()
    origins = {_normalize_base(_env(key)) for key in ("QC_API_BASE", "QUANTUMCOIN_API_BASE", "QC_NODE_API") if _env(key)}
    if len(origins) > 1:
        raise TelegramGameConnectionError("conflicting QuantumCoin API origins")
    configured = next(iter(origins), "")

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

    try:
        parsed = urlparse(base)
        port = parsed.port
        host = (parsed.hostname or "").lower()
    except ValueError:
        raise TelegramGameConnectionError("invalid QuantumCoin API origin") from None
    if not host or port == 0 or parsed.username is not None or parsed.password is not None or (
        parsed.path or parsed.params or parsed.query or parsed.fragment or any(char.isspace() for char in base)
    ):
        raise TelegramGameConnectionError("invalid QuantumCoin API origin")
    try:
        loopback = host == "localhost" or ip_address(host).is_loopback
    except ValueError:
        loopback = False
    if parsed.scheme != "https" and not (parsed.scheme == "http" and loopback):
        raise TelegramGameConnectionError("QuantumCoin API origin requires HTTPS or loopback HTTP")

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
        "testnetSimulatedMiningOnly": False,
        "realChainMiningAllowed": mode in {TG_CHAIN_MODE_TESTNET,TG_CHAIN_MODE_MAINNET},
        "requiresPinnedFreshGenesis": True,
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
