# handlers/moon.py — PTB 20.3, ASCII-only (final)
from datetime import datetime
from telegram import ReplyKeyboardMarkup, Update
from telegram.ext import ContextTypes, MessageHandler, CommandHandler, filters

from constants import SECTORS, USER_DAILY_SESSION_CAP
from utils.state import get_user, flood_guard, persist_user
from database.sqlite_store import session_start, session_finish
from handlers.core import main_keyboard


def _moon_kb() -> ReplyKeyboardMarkup:
    return ReplyKeyboardMarkup(
        [["Choose Sector", "Start Session"], ["Finish Session", "Back"]],
        resize_keyboard=True
    )


def _sector_kb() -> ReplyKeyboardMarkup:
    names = list(SECTORS.keys())
    rows = [names[i:i + 4] for i in range(0, len(names), 4)]
    rows.append(["Back"])
    return ReplyKeyboardMarkup(rows, resize_keyboard=True)


async def open_moon(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text(
        "Moon mining: Choose sector -> Start -> Finish after duration.",
        reply_markup=_moon_kb()
    )


async def choose_sector_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text("Select your sector:", reply_markup=_sector_kb())


async def select_sector(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    name = (update.message.text or "").strip()
    if name not in SECTORS:
        await update.message.reply_text("Invalid sector.")
        return
    if u.session_active:
        await update.message.reply_text("Session already active. Finish first.")
        return
    u.sector = name
    u.session_duration = SECTORS[name]["duration"]
    await update.message.reply_text(f"Sector set to {name}.", reply_markup=_moon_kb())


async def start_session(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    if not u.verified:
        await update.message.reply_text("Please verify first: use /verify.")
        return
    if u.sessions_today >= USER_DAILY_SESSION_CAP:
        await update.message.reply_text("Daily session limit reached. Try tomorrow.")
        return
    if u.session_active:
        await update.message.reply_text("Session already active. Use Finish Session.")
        return
    if not u.sector:
        await update.message.reply_text("Select a sector first.", reply_markup=_sector_kb())
        return

    cfg = SECTORS[u.sector]
    if u.energy < cfg["energy"]:
        await update.message.reply_text("Not enough energy. Use Daily to recover.")
        return

    u.energy -= cfg["energy"]
    u.session_start = datetime.utcnow()
    u.session_active = True
    u.session_duration = cfg["duration"]

    persist_user(u)
    session_start(u.user_id, u.sector, u.session_start.isoformat())

    await update.message.reply_text(
        f"Session started in {u.sector}. Duration: {u.session_duration}s. Energy: {u.energy}/100"
    )


async def finish_session(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    if not u.session_active or not u.session_start:
        await update.message.reply_text("No active session.")
        return

    elapsed = (datetime.utcnow() - u.session_start).total_seconds()
    if elapsed < u.session_duration:
        remaining = int(u.session_duration - elapsed)
        await update.message.reply_text(f"Too early. Wait {remaining}s more.")
        return

    cfg = SECTORS[u.sector]
    reward = round(cfg["base_qc"] * cfg["sector_mul"] * u.ship_mul, 3)
    u.qc = round(u.qc + reward, 3)
    u.session_active = False
    u.session_start = None
    u.sessions_today += 1

    persist_user(u)
    session_finish(u.user_id, u.sector, datetime.utcnow().isoformat(), reward)

    await update.message.reply_text(
        f"Session complete. +{reward} QC at {u.sector}. "
        f"Energy: {u.energy}/100 | QC: {u.qc} | "
        f"Sessions today: {u.sessions_today}/{USER_DAILY_SESSION_CAP}",
        reply_markup=main_keyboard()
    )


def wire(app):
    app.add_handler(CommandHandler("moon", open_moon))
    app.add_handler(MessageHandler(filters.Regex(r"^Moon$"), open_moon))
    app.add_handler(MessageHandler(filters.Regex(r"^Choose Sector$"), choose_sector_menu))

    names_pattern = "^(" + "|".join(map(lambda s: s.replace(" ", r"\ "), SECTORS.keys())) + ")$"
    app.add_handler(MessageHandler(filters.Regex(names_pattern), select_sector))
    app.add_handler(MessageHandler(filters.Regex(r"^Sector \d+$"), select_sector))

    app.add_handler(MessageHandler(filters.Regex(r"^Start Session$"), start_session))
    app.add_handler(MessageHandler(filters.Regex(r"^Finish Session$"), finish_session))
