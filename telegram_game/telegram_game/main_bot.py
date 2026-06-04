# telegram_game/main_bot.py

import logging
import random
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Final, Dict, Optional
import webbrowser

from telegram import (
    Update,
    InlineKeyboardMarkup,
    InlineKeyboardButton,
    WebAppInfo,  # LAUNCH button
)
from telegram.ext import (
    Application,
    CallbackQueryHandler,
    CommandHandler,
    ContextTypes,
)

from . import config  # get_bot_token, get_api_base, get_web_app_url

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    level=logging.INFO,
)
logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Callback data constants
# ---------------------------------------------------------------------------

CB_MAIN_MENU: Final[str] = "menu_main"
CB_MINE: Final[str] = "menu_mine"
CB_WALLET: Final[str] = "menu_wallet"
CB_LEADERBOARD: Final[str] = "menu_leaderboard"
CB_TGWT: Final[str] = "menu_tgwt"
CB_SETTINGS: Final[str] = "menu_settings"
CB_ABOUT: Final[str] = "menu_about"

# Yeni: Telegram'dan lokal dev UI açan buton
CB_LAUNCH_DEV: Final[str] = "launch_dev_ui"

# Madencilik için ek callback pattern'leri
CB_MINE_STATUS: Final[str] = "mine_status"
CB_MINE_CANCEL: Final[str] = "mine_cancel"
CB_MINE_NEW: Final[str] = "mine_new"

PREFIX_REGION_SELECT: Final[str] = "mine_region:"
PREFIX_REGION_CONFIRM: Final[str] = "mine_confirm:"


# New: Social & Watch & Earn
CB_SOCIAL: Final[str] = "menu_social"
CB_WATCH: Final[str] = "menu_watch"
CB_WATCH_CLAIM: Final[str] = "watch_claim"


# ---------------------------------------------------------------------------
# Mining configuration
# ---------------------------------------------------------------------------

@dataclass
class MiningRegion:
    key: str
    name: str
    icon: str
    duration_sec: int
    reward_min_qc: float
    reward_max_qc: float
    reward_min_tgwt: int
    reward_max_tgwt: int
    reward_min_xp: int
    reward_max_xp: int
    description: str


@dataclass
class Mission:
    user_id: int
    region_key: str
    region_name: str
    icon: str
    duration_sec: int
    reward_min_qc: float
    reward_max_qc: float
    reward_min_tgwt: int
    reward_max_tgwt: int
    reward_min_xp: int
    reward_max_xp: int
    started_at: datetime
    end_at: datetime
    status: str  # "active", "completed", "canceled"
    job_name: Optional[str] = None  # JobQueue name


# 3 main regions – first version
MINING_REGIONS: Dict[str, MiningRegion] = {
    "moon": MiningRegion(
        key="moon",
        name="Moon Orbit",
        icon="🌙",
        duration_sec=2 * 60,  # 2 minutes (short for tests)
        reward_min_qc=3.0,
        reward_max_qc=10.0,
        reward_min_tgwt=20,
        reward_max_tgwt=60,
        reward_min_xp=10,
        reward_max_xp=25,
        description=(
            "Low-risk, short mission. Small but safe QC and TGWT income."
        ),
    ),
    "asteroid": MiningRegion(
        key="asteroid",
        name="Asteroid Belt",
        icon="☄️",
        duration_sec=5 * 60,  # 5 minutes
        reward_min_qc=15.0,
        reward_max_qc=40.0,
        reward_min_tgwt=40,
        reward_max_tgwt=120,
        reward_min_xp=20,
        reward_max_xp=40,
        description=(
            "Medium risk and duration. Balanced rewards."
        ),
    ),
    "deep": MiningRegion(
        key="deep",
        name="Deep Space Mines",
        icon="🌀",
        duration_sec=10 * 60,  # 10 minutes
        reward_min_qc=40.0,
        reward_max_qc=90.0,
        reward_min_tgwt=100,
        reward_max_tgwt=260,
        reward_min_xp=35,
        reward_max_xp=70,
        description=(
            "High risk, long mission. High QC/TGWT and XP potential."
        ),
    ),
}


# ---------------------------------------------------------------------------
# Helpers: mission store
# ---------------------------------------------------------------------------

def _get_mission_store(context: ContextTypes.DEFAULT_TYPE) -> Dict[int, Mission]:
    """
    Global mission store for all users, in application.bot_data["missions"].
    """
    store = context.application.bot_data.get("missions")
    if store is None:
        store = {}
        context.application.bot_data["missions"] = store
    return store


def get_user_mission(context: ContextTypes.DEFAULT_TYPE, user_id: int) -> Optional[Mission]:
    return _get_mission_store(context).get(user_id)


def set_user_mission(context: ContextTypes.DEFAULT_TYPE, mission: Mission) -> None:
    store = _get_mission_store(context)
    store[mission.user_id] = mission


def clear_user_mission(context: ContextTypes.DEFAULT_TYPE, user_id: int) -> None:
    store = _get_mission_store(context)
    if user_id in store:
        del store[user_id]


# ---------------------------------------------------------------------------
# Main menu layout (text menu – fallback)
# ---------------------------------------------------------------------------

def build_main_menu_keyboard() -> InlineKeyboardMarkup:
    """
    GUI-like main menu buttons (text menu mode).
    """
    keyboard = [
        [
            InlineKeyboardButton("⛏ Start Mining", callback_data=CB_MINE),
        ],
        [
            InlineKeyboardButton("💼 Wallet", callback_data=CB_WALLET),
            InlineKeyboardButton("🏆 Leaderboard", callback_data=CB_LEADERBOARD),
        ],
        [
            InlineKeyboardButton("🎁 TGWT Rewards", callback_data=CB_TGWT),
        ],
        [
            InlineKeyboardButton("⭐ Social Missions", callback_data=CB_SOCIAL),
            InlineKeyboardButton("▶️ Watch & Earn", callback_data=CB_WATCH),
        ],
        [
            InlineKeyboardButton("⚙️ Settings", callback_data=CB_SETTINGS),
            InlineKeyboardButton("ℹ️ About", callback_data=CB_ABOUT),
        ],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_back_to_main_keyboard() -> InlineKeyboardMarkup:
    keyboard = [
        [InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU)],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_social_missions_keyboard() -> InlineKeyboardMarkup:
    """
    Social Missions screen with URL buttons for YouTube / Instagram / TikTok / Telegram.
    """
    youtube_url = getattr(config, "YOUTUBE_URL", "#")
    instagram_url = getattr(config, "INSTAGRAM_URL", "#")
    tiktok_url = getattr(config, "TIKTOK_URL", "#")
    telegram_url = getattr(config, "TELEGRAM_URL", "#")

    keyboard = [
        [InlineKeyboardButton("▶️ YouTube", url=youtube_url)],
        [InlineKeyboardButton("📸 Instagram", url=instagram_url)],
        [InlineKeyboardButton("🎵 TikTok", url=tiktok_url)],
        [InlineKeyboardButton("📢 Telegram", url=telegram_url)],
        [InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU)],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_watch_and_earn_keyboard() -> InlineKeyboardMarkup:
    """
    Watch & Earn screen: open external ad page + claim button.
    """
    watch_url = getattr(config, "WATCH_EARN_URL", "#")

    keyboard = [
        [InlineKeyboardButton("🌐 Open Ad Page", url=watch_url)],
        [InlineKeyboardButton("✅ I’ve Watched – Claim Reward", callback_data=CB_WATCH_CLAIM)],
        [InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU)],
    ]
    return InlineKeyboardMarkup(keyboard)


# ---------------------------------------------------------------------------
# Screen texts – Text-based UI (English)
# ---------------------------------------------------------------------------

def main_menu_text(user_name: Optional[str] = None) -> str:
    user_part = f", *{user_name}*" if user_name else ""
    return (
        f"🌌 *QuantumCoin Space Miner*{user_part}\n\n"
        "Welcome to the QuantumCoin space mining universe! 🚀\n\n"
        "From here you can control the entire game:\n\n"
        "• ⛏ Start mining missions\n"
        "• 💼 View your wallet and balances\n"
        "• 🏆 Check the global leaderboard\n"
        "• 🎁 See your TGWT game rewards\n"
        "• ⭐ Complete social missions for bonuses\n"
        "• ▶️ Watch & Earn extra rewards\n\n"
        "_Use the menu below to choose where to go._"
    )


def wallet_screen_text() -> str:
    # TODO: Later we will fetch real balances & address from QC node API.
    return (
        "💼 *Wallet*\n\n"
        "Here you will see your wallet inside the QuantumCoin universe:\n"
        "• On-chain QC balance\n"
        "• In-game TGWT token balance\n"
        "• Wallet address (copy / link)\n\n"
        "_In future updates, this screen will be connected to the real "
        "blockchain API._"
    )


def leaderboard_screen_text() -> str:
    # TODO: Later we will connect to a backend that calculates seasonal leaderboards.
    return (
        "🏆 *Leaderboard*\n\n"
        "This screen will show the top space miners with the highest scores.\n\n"
        "• Seasonal Top 10\n"
        "• Total QC mined\n"
        "• Special seasonal rewards\n\n"
        "_This is a demo text for now. Real data will come from the game backend._"
    )


def tgwt_screen_text() -> str:
    # TODO: TGWT balances will come from smart contract or off-chain system.
    return (
        "🎁 *TGWT Rewards*\n\n"
        "This screen will show the TGWT tokens you earn from the Telegram game.\n\n"
        "• Daily / weekly mission rewards\n"
        "• TGWT → QC conversion panel\n"
        "• Special NFT & seasonal rewards\n\n"
        "_For now, this is just a UI skeleton. Backend will be added later._"
    )


def settings_screen_text() -> str:
    # TODO: Language, notifications, and mode will be saved in DB later.
    return (
        "⚙️ *Settings*\n\n"
        "Here you will customize your game experience:\n"
        "• Language: English (for now)\n"
        "• Notification preferences\n"
        "• Game mode (Relaxed / Competitive)\n\n"
        "_In the first version we can start with simple settings and expand later._"
    )


def about_screen_text() -> str:
    return (
        "ℹ️ *About*\n\n"
        "The QuantumCoin Telegram game is a space-mining themed mini game "
        "connected to the real QC blockchain.\n\n"
        "Goals:\n"
        "• Your mining missions and rewards will be written on-chain\n"
        "• Mobile / PC / Telegram will share the same economy\n"
        "• Seasonal events and special NFTs will be available\n\n"
        "_This version is only the UI skeleton. Blockchain calls will be "
        "added step by step._"
    )


def social_missions_text() -> str:
    return (
        "⭐ *Social Missions*\n\n"
        "Support the QuantumCoin ecosystem and earn extra TGWT:\n\n"
        "• Subscribe to our YouTube channel\n"
        "• Follow us on Instagram\n"
        "• Follow us on TikTok\n"
        "• Join our official Telegram channel\n\n"
        "Tap a platform icon below to open the page.\n"
        "More advanced follow & complete missions will be added later."
    )


def watch_and_earn_text() -> str:
    return (
        "▶️ *Watch & Earn*\n\n"
        "Watch ads and partner content to earn extra TGWT.\n\n"
        "How it works:\n"
        "1) Tap “Open Ad Page”\n"
        "2) Watch the content on our website (with mixed ads, similar to Google AdSense)\n"
        "3) Come back here and tap “I’ve Watched – Claim Reward”\n"
        "4) Receive your TGWT bonus (subject to daily limits)\n\n"
        "_In this early version we only show the UI; real reward logic and checks "
        "will be added later._"
    )


# ---------------------------------------------------------------------------
# Mining screen: text & keyboards
# ---------------------------------------------------------------------------

def mining_home_text(active_mission: Optional[Mission]) -> str:
    if active_mission is None:
        return (
            "⛏ *Mining Panel*\n\n"
            "Send your spaceship to different mining regions to perform real-time mining missions.\n\n"
            "1️⃣ Choose a region\n"
            "2️⃣ Start the mission\n"
            "3️⃣ When time is over, claim your rewards\n\n"
            "_Choose a mining region below to begin:_"
        )
    else:
        now = datetime.now(timezone.utc)
        remaining = max((active_mission.end_at - now).total_seconds(), 0)
        minutes = int(remaining // 60)
        seconds = int(remaining % 60)
        return (
            "⛏ *Mining Panel*\n\n"
            f"{active_mission.icon} You already have an active mission in *{active_mission.region_name}*.\n\n"
            f"⏳ Remaining time: *{minutes} min {seconds} sec*\n"
            "_You cannot start a new mission until this one is finished._"
        )


def build_mining_region_keyboard() -> InlineKeyboardMarkup:
    keyboard = [
        [
            InlineKeyboardButton("🌙 Moon Orbit", callback_data=f"{PREFIX_REGION_SELECT}moon"),
        ],
        [
            InlineKeyboardButton("☄️ Asteroid Belt", callback_data=f"{PREFIX_REGION_SELECT}asteroid"),
        ],
        [
            InlineKeyboardButton("🌀 Deep Space Mines", callback_data=f"{PREFIX_REGION_SELECT}deep"),
        ],
        [
            InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU),
        ],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_mining_active_keyboard() -> InlineKeyboardMarkup:
    keyboard = [
        [
            InlineKeyboardButton("📊 Mission Status", callback_data=CB_MINE_STATUS),
        ],
        [
            InlineKeyboardButton("🚫 Cancel Mission", callback_data=CB_MINE_CANCEL),
        ],
        [
            InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU),
        ],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_mission_confirm_keyboard(region_key: str) -> InlineKeyboardMarkup:
    keyboard = [
        [
            InlineKeyboardButton("🚀 Start Mission", callback_data=f"{PREFIX_REGION_CONFIRM}{region_key}"),
        ],
        [
            InlineKeyboardButton("❌ Cancel", callback_data=CB_MINE),
        ],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_mission_finished_keyboard() -> InlineKeyboardMarkup:
    keyboard = [
        [
            InlineKeyboardButton("⛏ Choose New Mission", callback_data=CB_MINE_NEW),
        ],
        [
            InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU),
        ],
    ]
    return InlineKeyboardMarkup(keyboard)


def build_mission_status_keyboard() -> InlineKeyboardMarkup:
    keyboard = [
        [
            InlineKeyboardButton("⬅️ Back to Mining Panel", callback_data=CB_MINE),
        ],
        [
            InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU),
        ],
    ]
    return InlineKeyboardMarkup(keyboard)


def region_detail_text(region: MiningRegion) -> str:
    minutes = region.duration_sec // 60
    return (
        f"{region.icon} *{region.name} Mission*\n\n"
        f"⏳ Duration: *{minutes} minutes*\n"
        f"💰 QC range: *{region.reward_min_qc:.0f} – {region.reward_max_qc:.0f} QC*\n"
        f"🎟 TGWT range: *{region.reward_min_tgwt} – {region.reward_max_tgwt}*\n"
        f"⭐ XP range: *{region.reward_min_xp} – {region.reward_max_xp}*\n\n"
        f"{region.description}\n\n"
        "_Do you want to send your ship on this mission?_"
    )


def mission_status_text(mission: Mission) -> str:
    now = datetime.now(timezone.utc)
    remaining = max((mission.end_at - now).total_seconds(), 0)
    minutes = int(remaining // 60)
    seconds = int(remaining % 60)

    progress_total = mission.duration_sec
    done = min(max(progress_total - remaining, 0), progress_total)
    if progress_total > 0:
        pct = int(done * 100 / progress_total)
    else:
        pct = 0

    # Simple text progress bar
    bar_len = 10
    filled_len = int(bar_len * pct / 100)
    bar = "█" * filled_len + "▒" * (bar_len - filled_len)

    return (
        f"{mission.icon} *Active Mission: {mission.region_name}*\n\n"
        f"⏱ Progress: `{bar}` *{pct}%*\n"
        f"⏳ Remaining time: *{minutes} min {seconds} sec*\n\n"
        "_When the mission is complete, you will receive a separate message with your rewards._"
    )


# ---------------------------------------------------------------------------
# Handlers – Commands (/start, /help)
# ---------------------------------------------------------------------------

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /start → Şimdilik lokal geliştirme modu.
    - 'Launch Dev UI (local)' butonu: Bu botun çalıştığı PC'de tarayıcı açar
    - 'Metin Menü (Yedek)': Klasik text tabanlı menü
    """
    user = update.effective_user
    logger.info("User %s (%s) used /start", user.id, user.username)

    keyboard = InlineKeyboardMarkup(
        [
            [
                InlineKeyboardButton(
                    text="🚀 Launch Dev UI (local)",
                    callback_data=CB_LAUNCH_DEV,  # yeni callback
                )
            ],
            [
                InlineKeyboardButton(
                    text="📋 Metin Menü (Yedek)",
                    callback_data=CB_MAIN_MENU,
                )
            ],
        ]
    )

    first_name = user.first_name if user else ""
    text = (
        "🌌 *QuantumCoin Space Miner*\n\n"
        f"Hoş geldin {first_name}!\n\n"
        "Şu anda oyunun arayüzünü *lokal olarak* senin bilgisayarında "
        "geliştiriyoruz.\n\n"
        "• 🚀 *Launch Dev UI (local)* → Bu PC'de tarayıcıda oyun penceresini açar\n"
        "• 📋 *Metin Menü (Yedek)* → Telegram içindeki klasik menü arayüzü\n\n"
        "Aşağıdaki butonlardan birini seç."
    )

    if update.message:
        await update.message.reply_markdown(text, reply_markup=keyboard)
    elif update.callback_query and update.callback_query.message:
        await update.callback_query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )



async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    await update.message.reply_text(
        "This bot is the Telegram gateway to the QuantumCoin space mining game.\n"
        "Use /start and tap *Launch QuantumCoin Game* to open the full-screen game.\n"
        "Or use the Text Menu (fallback) to control everything with inline buttons."
    )


# ---------------------------------------------------------------------------
# Command shortcuts (text menu – /mine, /wallet, ...)
# ---------------------------------------------------------------------------

async def mine_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /mine → open Mining Panel (text menu mode).
    """
    await handle_mining_home(update, context)


async def wallet_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /wallet → Wallet screen (text menu mode).
    """
    text = wallet_screen_text()
    keyboard = build_back_to_main_keyboard()

    if update.message:
        await update.message.reply_markdown(
            text,
            reply_markup=keyboard,
        )


async def leaderboard_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /leaderboard → Leaderboard screen (text menu mode).
    """
    text = leaderboard_screen_text()
    keyboard = build_back_to_main_keyboard()

    if update.message:
        await update.message.reply_markdown(
            text,
            reply_markup=keyboard,
        )


async def tgwt_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /tgwt → TGWT Rewards screen (text menu mode).
    """
    text = tgwt_screen_text()
    keyboard = build_back_to_main_keyboard()

    if update.message:
        await update.message.reply_markdown(
            text,
            reply_markup=keyboard,
        )


async def settings_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /settings → Settings screen (text menu mode).
    """
    text = settings_screen_text()
    keyboard = build_back_to_main_keyboard()

    if update.message:
        await update.message.reply_markdown(
            text,
            reply_markup=keyboard,
        )


async def about_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /about → About screen (text menu mode).
    """
    text = about_screen_text()
    keyboard = build_back_to_main_keyboard()

    if update.message:
        await update.message.reply_markdown(
            text,
            reply_markup=keyboard,
        )


# ---------------------------------------------------------------------------
# Handlers – Main menu navigation (text menu)
# ---------------------------------------------------------------------------

async def show_main_menu(
    update: Update,
    context: ContextTypes.DEFAULT_TYPE,
    query_message: bool = True,
) -> None:
    user = update.effective_user
    text = main_menu_text(user_name=user.first_name if user else None)
    keyboard = build_main_menu_keyboard()

    if query_message and update.callback_query and update.callback_query.message:
        await update.callback_query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )
    else:
        if update.effective_chat:
            await update.effective_chat.send_message(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )


# ---------------------------------------------------------------------------
# Handlers – Mining flow
# ---------------------------------------------------------------------------

async def handle_mining_home(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    Called from CB_MINE / CB_MINE_NEW callbacks or /mine command.
    If there is an active mission, show its status; otherwise show region selection.
    """
    user = update.effective_user
    query = update.callback_query
    mission = get_user_mission(context, user.id)

    if mission and mission.status == "active":
        text = mining_home_text(mission)
        keyboard = build_mining_active_keyboard()
    else:
        text = mining_home_text(None)
        keyboard = build_mining_region_keyboard()

    if query and query.message:
        await query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )
    elif update.message:
        await update.message.reply_markdown(
            text,
            reply_markup=keyboard,
        )


async def handle_mining_region_select(update: Update, context: ContextTypes.DEFAULT_TYPE, region_key: str) -> None:
    """
    When a mining region is selected (Moon, Asteroid, Deep space),
    show mission details + Start / Cancel.
    """
    user = update.effective_user
    mission = get_user_mission(context, user.id)
    query = update.callback_query

    if mission and mission.status == "active":
        text = (
            "⚠️ You already have an active mission.\n\n"
            "You can view its status or cancel it below."
        )
        keyboard = build_mining_active_keyboard()
        if query and query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    region = MINING_REGIONS.get(region_key)
    if not region:
        if query and query.message:
            await query.message.edit_text(
                "Selected mining region was not found. Please try again.",
                reply_markup=build_mining_region_keyboard(),
            )
        return

    text = region_detail_text(region)
    keyboard = build_mission_confirm_keyboard(region_key)

    if query and query.message:
        await query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )


async def handle_mining_start(update: Update, context: ContextTypes.DEFAULT_TYPE, region_key: str) -> None:
    """
    Called when the user confirms 'Start Mission'.
    Creates a Mission and schedules a JobQueue callback.
    """
    user = update.effective_user
    query = update.callback_query

    existing = get_user_mission(context, user.id)
    if existing and existing.status == "active":
        text = (
            "⚠️ You already have an active mission.\n\n"
            "You cannot start another one until it finishes."
        )
        keyboard = build_mining_active_keyboard()
        if query and query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    region = MINING_REGIONS.get(region_key)
    if not region:
        if query and query.message:
            await query.message.edit_text(
                "Selected mining region was not found. Please try again.",
                reply_markup=build_mining_region_keyboard(),
            )
        return

    now = datetime.now(timezone.utc)
    end_at = now + timedelta(seconds=region.duration_sec)

    mission = Mission(
        user_id=user.id,
        region_key=region.key,
        region_name=region.name,
        icon=region.icon,
        duration_sec=region.duration_sec,
        reward_min_qc=region.reward_min_qc,
        reward_max_qc=region.reward_max_qc,
        reward_min_tgwt=region.reward_min_tgwt,
        reward_max_tgwt=region.reward_max_tgwt,
        reward_min_xp=region.reward_min_xp,
        reward_max_xp=region.reward_max_xp,
        started_at=now,
        end_at=end_at,
        status="active",
        job_name=None,
    )

    job_name = f"mission_complete_{user.id}"
    mission.job_name = job_name
    context.job_queue.run_once(
        mission_complete_job,
        when=region.duration_sec,
        name=job_name,
        data={"user_id": user.id},
    )

    set_user_mission(context, mission)

    minutes = region.duration_sec // 60
    text = (
        "🚀 *Mission Started!*\n\n"
        f"Your ship is heading to {region.icon} *{region.name}*.\n\n"
        f"⏳ Duration: *{minutes} minutes*\n\n"
        "_During this time you cannot start another mission. "
        "When it finishes, you will receive a message with your rewards._"
    )
    keyboard = build_mining_active_keyboard()

    if query and query.message:
        await query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )


async def handle_mining_status(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    '📊 Mission Status' button.
    """
    user = update.effective_user
    query = update.callback_query
    mission = get_user_mission(context, user.id)

    if not mission or mission.status != "active":
        text = (
            "You do not have an active mission right now.\n\n"
            "_Use the button below to start a new mission._"
        )
        keyboard = InlineKeyboardMarkup(
            [
                [InlineKeyboardButton("⛏ Choose New Mission", callback_data=CB_MINE_NEW)],
                [InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU)],
            ]
        )
    else:
        text = mission_status_text(mission)
        keyboard = build_mission_status_keyboard()

    if query and query.message:
        await query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )


async def handle_mining_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    '🚫 Cancel Mission' button.
    Cancels the active mission and removes the JobQueue job.
    """
    user = update.effective_user
    query = update.callback_query
    mission = get_user_mission(context, user.id)

    if not mission or mission.status != "active":
        text = "There is no active mission to cancel."
        keyboard = build_mining_region_keyboard()
        if query and query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # Remove JobQueue job
    if mission.job_name:
        jobs = context.job_queue.get_jobs_by_name(mission.job_name)
        for job in jobs:
            job.schedule_removal()

    mission.status = "canceled"
    clear_user_mission(context, user.id)

    text = (
        "🚫 *Mission Canceled.*\n\n"
        "You called your ship back. You will not receive rewards from this mission.\n\n"
        "_You can start a new mission if you want._"
    )
    keyboard = InlineKeyboardMarkup(
        [
            [InlineKeyboardButton("⛏ Choose New Mission", callback_data=CB_MINE_NEW)],
            [InlineKeyboardButton("⬅️ Back to Main Menu", callback_data=CB_MAIN_MENU)],
        ]
    )

    if query and query.message:
        await query.message.edit_text(
            text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )


# ---------------------------------------------------------------------------
# JobQueue callback – mission completion
# ---------------------------------------------------------------------------

async def mission_complete_job(context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    Called by JobQueue when mission time is over.
    Sends reward message to the user.
    """
    job = context.job
    if not job or not job.data:
        return

    user_id = job.data.get("user_id")
    if not user_id:
        return

    store = _get_mission_store(context)
    mission = store.get(user_id)

    if not mission or mission.status != "active":
        return

    # Random rewards
    qc = random.uniform(mission.reward_min_qc, mission.reward_max_qc)
    tgwt = random.randint(mission.reward_min_tgwt, mission.reward_max_tgwt)
    xp = random.randint(mission.reward_min_xp, mission.reward_max_xp)

    mission.status = "completed"
    clear_user_mission(context, user_id)

    # TODO: Here we will call real blockchain API to write QC/TGWT/XP.

    text = (
        "✅ *Mission Completed!*\n\n"
        f"{mission.icon} Your *{mission.region_name}* mission finished successfully.\n\n"
        "You earned:\n"
        f"• 💰 QC: *{qc:.2f}*\n"
        f"• 🎟 TGWT: *{tgwt}*\n"
        f"• ⭐ XP: *{xp}*\n\n"
        "_Your balances and stats have been updated (on-chain integration will be added later)._"
    )

    keyboard = build_mission_finished_keyboard()

    try:
        await context.bot.send_message(
            chat_id=user_id,
            text=text,
            reply_markup=keyboard,
            parse_mode="Markdown",
        )
    except Exception as e:
        logger.error("mission_complete_job send_message error: %s", e)
async def handle_launch_dev_ui(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    Telegram'dan gelen buton ile bu botun çalıştığı PC'de
    lokal dev arayüzünü (tarayıcıda) açar.
    Sadece bu bilgisayarda işe yarar (sunucuda çalışıyorsa tarayıcı orada açılır).
    """
    query = update.callback_query
    if query:
        # Küçük loading animasyonu kapanması için
        await query.answer()

    # config'te DEV_UI_URL tanımlıysa onu kullan, yoksa varsayılan:
    dev_url = getattr(config, "DEV_UI_URL", "http://127.0.0.1:8000")

    try:
        webbrowser.open(dev_url)
        msg = (
            "🖥 Local Dev UI bu makinede açılıyor:\n"
            f"{dev_url}\n\n"
            "Bu adreste bir geliştirme sunucusu (veya basit bir index.html) "
            "çalıştığından emin ol."
        )
    except Exception as e:
        logger.error("Failed to open local dev UI: %s", e)
        msg = (
            "⚠️ Lokal Dev UI açılmaya çalışıldı ama bu makinede bir hata oluştu.\n"
            f"URL: {dev_url}"
        )

    # Kullanıcıya kısa bilgi mesajı
    if query and query.message:
        await query.message.reply_text(msg)


# ---------------------------------------------------------------------------
# Handler – Callback router
# ---------------------------------------------------------------------------

async def handle_menu_callback(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    query = update.callback_query
    if not query:
        return

    data = query.data or ""
    await query.answer()
    logger.info("Callback data: %s", data)

    # Main menu (text)
    if data == CB_MAIN_MENU:
        await show_main_menu(update, context, query_message=True)
        return
    # Lokal Dev UI'yi aç (PC'de tarayıcı penceresi)
    if data == CB_LAUNCH_DEV:
        await handle_launch_dev_ui(update, context)
        return

    # Mining home / new mission
    if data in (CB_MINE, CB_MINE_NEW):
        await handle_mining_home(update, context)
        return

    # Mining: status
    if data == CB_MINE_STATUS:
        await handle_mining_status(update, context)
        return

    # Mining: cancel
    if data == CB_MINE_CANCEL:
        await handle_mining_cancel(update, context)
        return

    # Mining: region select
    if data.startswith(PREFIX_REGION_SELECT):
        region_key = data[len(PREFIX_REGION_SELECT):]
        await handle_mining_region_select(update, context, region_key)
        return

    # Mining: confirm start
    if data.startswith(PREFIX_REGION_CONFIRM):
        region_key = data[len(PREFIX_REGION_CONFIRM):]
        await handle_mining_start(update, context, region_key)
        return

    # Wallet
    if data == CB_WALLET:
        text = wallet_screen_text()
        keyboard = build_back_to_main_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # Leaderboard
    if data == CB_LEADERBOARD:
        text = leaderboard_screen_text()
        keyboard = build_back_to_main_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # TGWT
    if data == CB_TGWT:
        text = tgwt_screen_text()
        keyboard = build_back_to_main_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # Social Missions
    if data == CB_SOCIAL:
        text = social_missions_text()
        keyboard = build_social_missions_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # Watch & Earn screen
    if data == CB_WATCH:
        text = watch_and_earn_text()
        keyboard = build_watch_and_earn_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # Watch & Earn – claim (placeholder)
    if data == CB_WATCH_CLAIM:
        if query.message:
            await query.message.edit_text(
                "Watch & Earn rewards are not implemented yet.\n"
                "In a future update, this button will grant TGWT after basic checks.\n\n"
                "Use /start to open the main menu again.",
            )
        return

    # Settings
    if data == CB_SETTINGS:
        text = settings_screen_text()
        keyboard = build_back_to_main_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # About
    if data == CB_ABOUT:
        text = about_screen_text()
        keyboard = build_back_to_main_keyboard()
        if query.message:
            await query.message.edit_text(
                text,
                reply_markup=keyboard,
                parse_mode="Markdown",
            )
        return

    # Unknown → back to main menu
    await show_main_menu(update, context, query_message=True)


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    """
    Entry point. Run with:
    python -m telegram_game.main_bot
    """
    token = config.get_bot_token().strip()
    if not token:
        raise RuntimeError(
            "Telegram bot token not found. Please check TELEGRAM_BOT_TOKEN in .env "
            "or _DEFAULT_BOT_TOKEN in config.py."
        )

    qc_api_base = config.get_api_base()
    web_app_url = config.get_web_app_url()
    logger.info("Using QC_API_BASE = %s", qc_api_base)
    logger.info("Using WEB_APP_URL = %s", web_app_url)

    application = Application.builder().token(token).build()
    application.bot_data["QC_API_BASE"] = qc_api_base
    application.bot_data["WEB_APP_URL"] = web_app_url

    # Commands
    application.add_handler(CommandHandler("start", start))
    application.add_handler(CommandHandler("help", help_command))

    # Game commands (text menu)
    application.add_handler(CommandHandler("mine", mine_command))
    application.add_handler(CommandHandler("wallet", wallet_command))
    application.add_handler(CommandHandler("leaderboard", leaderboard_command))
    application.add_handler(CommandHandler("tgwt", tgwt_command))
    application.add_handler(CommandHandler("settings", settings_command))
    application.add_handler(CommandHandler("about", about_command))

    # Menu callbacks (inline buttons)
    application.add_handler(CallbackQueryHandler(handle_menu_callback))

    logger.info("QuantumCoin Telegram Game bot is starting...")
    application.run_polling(allowed_updates=Update.ALL_TYPES)


if __name__ == "__main__":
    main()
