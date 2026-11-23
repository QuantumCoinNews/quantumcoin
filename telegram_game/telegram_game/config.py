# telegram_game/config.py
import os
from pathlib import Path

# Optional .env loader. If python-dotenv is not installed, this safely does nothing.
try:
    from dotenv import load_dotenv
    if os.getenv("QC_GAME_LOAD_DOTENV", "1") == "1":
        load_dotenv(Path(__file__).resolve().parents[1] / ".env", override=False)
except Exception:
    pass

from .shared_environment import require_game_environment
require_game_environment(os.environ)

# --- Telegram bot ---
# SECURITY:
# Never hardcode the real bot token in this file.
# Set TELEGRAM_BOT_TOKEN in .env or Windows environment variables.
BOT_TOKEN = (os.getenv("TELEGRAM_BOT_TOKEN") or os.getenv("QC_TELEGRAM_BOT_TOKEN")
             or os.getenv("TELEGRAM_GAME_BOT_TOKEN") or "").strip()

# --- QuantumCoin API ---
from .testnet_connection_v1 import (
    get_qc_api_base_for_mode_v1, telegram_game_connection_info_v1,
    assert_telegram_game_connection_safe_v1,
)
QC_API_BASE = get_qc_api_base_for_mode_v1()

# --- Social media links ---
YOUTUBE_URL   = os.getenv("QC_YOUTUBE_URL", "https://www.youtube.com/@QuantumCoinHQ").strip()
X_URL         = os.getenv("QC_X_URL", "https://x.com/QuantumCoinQC").strip()
INSTAGRAM_URL = os.getenv("QC_INSTAGRAM_URL", "https://www.instagram.com/quantumcoinnews").strip()
TIKTOK_URL    = os.getenv("QC_TIKTOK_URL", "https://www.tiktok.com/@quantumcoin21").strip()
TELEGRAM_URL  = os.getenv("QC_TELEGRAM_URL", "https://web.telegram.org/a/#-1002870924021").strip()

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


def get_web_app_url() -> str:
    """
    Telegram WebApp fullscreen game URL.
    """
    return WEB_APP_URL


# One dynamic resolver per operation; no order-dependent overrides.
def get_api_base():
    return get_qc_api_base_for_mode_v1()


def get_connection_info_v1():
    return telegram_game_connection_info_v1()


def assert_connection_safe_v1():
    return assert_telegram_game_connection_safe_v1()
