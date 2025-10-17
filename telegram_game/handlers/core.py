# handlers/core.py — PTB 20.3, ASCII-only (final)
from telegram import ReplyKeyboardMarkup, Update
from telegram.ext import ContextTypes, CommandHandler, MessageHandler, filters

from constants import USER_DAILY_SESSION_CAP
from utils.state import get_user, flood_guard, persist_user
from database.sqlite_store import insert_referral


def main_keyboard() -> ReplyKeyboardMarkup:
    rows = [
        ["Moon", "Social"],
        ["Leaderboard", "Referral"],
        ["Profile", "Wallet"],
        ["Daily", "Balance"],
    ]
    return ReplyKeyboardMarkup(rows, resize_keyboard=True)


def hud_text(u) -> str:
    return (
        "Quantum Mining HUD\n"
        "Welcome to Quantum Mining!\n"
        f"QC: {u.qc}\n"
        f"Energy: {u.energy}/100\n"
        f"Level: {u.level}\n"
        f"Ship: {u.ship} (x{u.ship_mul})\n"
        f"TGWT: {u.tgwt}\n"
        f"Sector: {u.sector or '-'}\n"
        f"Session: {'active' if u.session_active else '-'}\n"
    )


async def cmd_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    # Referral param parse
    txt = (getattr(update.message, "text", "") or "").strip()
    if txt.startswith("/start ") and "ref_" in txt:
        try:
            ref_id = int(txt.split("ref_")[1])
            me = get_user(update)
            if ref_id != me.user_id:
                insert_referral(ref_id, me.user_id)
        except Exception:
            pass

    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text(hud_text(u), reply_markup=main_keyboard())


async def cmd_help(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text(
        "Commands:\n"
        "/start - show HUD\n/menu - show main menu\n"
        "Moon -> Choose Sector -> Start -> Finish\n"
        "Social -> Verify, Claim TGWT\n"
        "Daily - energy refill\nLeaderboard - top by QC"
    )


async def cmd_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text("Main menu:", reply_markup=main_keyboard())


async def btn_balance(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text(hud_text(u), reply_markup=main_keyboard())


async def btn_daily(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    before = u.energy
    u.energy = min(100, u.energy + 25)
    persist_user(u)
    gained = u.energy - before
    await update.message.reply_text(f"Energy +{gained}. Now {u.energy}/100.", reply_markup=main_keyboard())


async def btn_profile(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text(
        f"Profile\nQC: {u.qc}\nTGWT: {u.tgwt}\nEnergy: {u.energy}/100\nLevel: {u.level}"
    )


async def btn_wallet(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text("Wallet: not linked yet.", reply_markup=main_keyboard())


def wire_commands(app):
    # Commands
    app.add_handler(CommandHandler("start", cmd_start), group=-90)
    app.add_handler(CommandHandler("help",  cmd_help),  group=0)
    app.add_handler(CommandHandler("menu",  cmd_menu),  group=0)
    app.add_handler(CommandHandler("balance", btn_balance), group=5)
    app.add_handler(CommandHandler("daily",   btn_daily),   group=5)

    # Buttons
    app.add_handler(MessageHandler(filters.Regex(r"^Balance$"),   btn_balance), group=10)
    app.add_handler(MessageHandler(filters.Regex(r"^Daily$"),     btn_daily),   group=10)
    app.add_handler(MessageHandler(filters.Regex(r"^Profile$"),   btn_profile), group=10)
    app.add_handler(MessageHandler(filters.Regex(r"^Wallet$"),    btn_wallet),  group=10)
