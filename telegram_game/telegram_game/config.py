# telegram_game/config.py
import os

# Optional .env loader. If python-dotenv is not installed, this safely does nothing.
try:
    from dotenv import load_dotenv
    load_dotenv()
except Exception:
    pass

# --- Telegram bot ---
# SECURITY:
# Never hardcode the real bot token in this file.
# Set TELEGRAM_BOT_TOKEN in .env or Windows environment variables.
BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN", "").strip()

# --- QuantumCoin API ---
QC_API_BASE = (
    os.getenv("QC_API_BASE")
    or os.getenv("QUANTUMCOIN_API_BASE")
    or os.getenv("QC_NODE_API")
    or "http://127.0.0.1:8082"
).strip()

# --- Social media links ---
YOUTUBE_URL   = os.getenv("QC_YOUTUBE_URL", "https://www.youtube.com/@QuantumCoinHQ").strip()
X_URL         = os.getenv("QC_X_URL", "https://x.com/QuantumCoinQC").strip()
INSTAGRAM_URL = os.getenv("QC_INSTAGRAM_URL", "https://www.instagram.com/quantumcoinnews").strip()
TIKTOK_URL    = os.getenv("QC_TIKTOK_URL", "https://www.tiktok.com/@quantumcoin21").strip()
TELEGRAM_URL  = os.getenv("QC_TELEGRAM_URL", "https://t.me/QuantumCoinNews").strip()

# --- Watch & Earn ---
WATCH_EARN_URL = os.getenv("QC_WATCH_EARN_URL", "https://qcnetwork.ai/ads").strip()

# --- Telegram WebApp game URL ---
WEB_APP_URL = os.getenv("QC_WEB_APP_URL", "https://qcnetwork.ai/game").strip()


def get_bot_token() -> str:
    """
    Telegram bot token.
    Must be provided by TELEGRAM_BOT_TOKEN.
    """
    return BOT_TOKEN


def get_api_base() -> str:
    """
    QuantumCoin API / node base URL.
    """
    return QC_API_BASE


def get_web_app_url() -> str:
    """
    Telegram WebApp fullscreen game URL.
    """
    return WEB_APP_URL


# QC_TG_TESTNET_CONNECTION_V1
# Centralized Telegram Game -> QuantumCoin API connection separation.
try:
    from telegram_game.testnet_connection_v1 import (
        get_qc_api_base_for_mode_v1,
        telegram_game_connection_info_v1,
        assert_telegram_game_connection_safe_v1,
    )

    QC_API_BASE = get_qc_api_base_for_mode_v1()
except Exception:
    # Keep import-time safety for tooling; runtime checks are performed by callers/tests.
    pass


def get_connection_info_v1():
    from telegram_game.testnet_connection_v1 import telegram_game_connection_info_v1

    return telegram_game_connection_info_v1()


def assert_connection_safe_v1():
    from telegram_game.testnet_connection_v1 import assert_telegram_game_connection_safe_v1

    return assert_telegram_game_connection_safe_v1()

# QC_TG_TESTNET_CONNECTION_V1_RUNTIME_BRIDGE
# Override get_api_base so runtime callers always use the centralized testnet/local guard.
def get_api_base():
    from telegram_game.testnet_connection_v1 import get_qc_api_base_for_mode_v1

    return get_qc_api_base_for_mode_v1()


def get_connection_info_v1():
    from telegram_game.testnet_connection_v1 import telegram_game_connection_info_v1

    return telegram_game_connection_info_v1()


def assert_connection_safe_v1():
    from telegram_game.testnet_connection_v1 import assert_telegram_game_connection_safe_v1

    return assert_telegram_game_connection_safe_v1()
