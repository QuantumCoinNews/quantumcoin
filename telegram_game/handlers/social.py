# handlers/social.py — PTB 20.3, ASCII-only
import random
import string
from telegram import ReplyKeyboardMarkup, Update
from telegram.ext import ContextTypes, CommandHandler, MessageHandler, filters

from handlers.core import main_keyboard  # Back -> show main keyboard
from utils.state import get_user, flood_guard, persist_user
from utils.tgwt_pool import try_claim_tgwt
from constants import SOCIAL_TASKS


def _social_kb() -> ReplyKeyboardMarkup:
    return ReplyKeyboardMarkup(
        [["Verify", "Claim TGWT"], ["Back"]],
        resize_keyboard=True
    )


def _tasks_kb() -> ReplyKeyboardMarkup:
    names = list(SOCIAL_TASKS.keys())  # e.g., Follow YouTube, Follow X, ...
    rows = [names[i:i + 2] for i in range(0, len(names), 2)]
    rows.append(["Back"])
    return ReplyKeyboardMarkup(rows, resize_keyboard=True)


def _mk_code(n: int = 5) -> str:
    alphabet = string.ascii_uppercase + string.digits
    return "".join(random.choice(alphabet) for _ in range(n))


# -------- screens --------

async def open_social(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text(
        "Social menu:\n- Verify to unlock mining\n- Claim TGWT for social tasks",
        reply_markup=_social_kb()
    )


async def verify_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    code = _mk_code(5)
    u.verify_code = code
    await update.message.reply_text(
        f"Human check. Reply exactly with:\nCODE {code}\n(ASCII only)",
        reply_markup=_social_kb()
    )


async def verify_check(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    txt = (update.message.text or "").strip()
    # Only handle messages that start with CODE ...
    if not txt.upper().startswith("CODE"):
        return  # let other handlers try
    parts = txt.split()
    if len(parts) != 2:
        await update.message.reply_text("Wrong format. Use /verify to get a new one.", reply_markup=_social_kb())
        return
    if parts[1].upper() == (u.verify_code or "").upper():
        u.verified = True
        persist_user(u)
        await update.message.reply_text("Verification OK. You can start mining now.", reply_markup=_social_kb())
    else:
        await update.message.reply_text("Wrong code. Use /verify to get a new one.", reply_markup=_social_kb())


async def claim_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    await update.message.reply_text("Choose a social task to claim TGWT:", reply_markup=_tasks_kb())


async def claim_task(update: Update, context: ContextTypes.DEFAULT_TYPE):
    u = get_user(update)
    if not flood_guard(u):
        return
    task = (update.message.text or "").strip()
    ok, amount, msg = try_claim_tgwt(u.user_id, task)
    if ok:
        u.tgwt += amount
        persist_user(u)
        await update.message.reply_text(f"{msg}. Your TGWT: {u.tgwt}", reply_markup=_tasks_kb())
    else:
        await update.message.reply_text(msg, reply_markup=_tasks_kb())


# -------- wiring --------

def wire(app):
    # Open social
    app.add_handler(CommandHandler("social", open_social), group=10)
    app.add_handler(MessageHandler(filters.Regex(r"^Social$"), open_social), group=10)

    # Verify flow
    app.add_handler(CommandHandler("verify", verify_start), group=10)
    app.add_handler(MessageHandler(filters.Regex(r"^Verify$"), verify_start), group=10)
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, verify_check), group=20)

    # Claim TGWT
    app.add_handler(MessageHandler(filters.Regex(r"^Claim TGWT$"), claim_menu), group=10)

    # Social task buttons (exact match any of the task names)
    names_pattern = "^(" + "|".join(map(lambda s: s.replace(" ", r"\ "), SOCIAL_TASKS.keys())) + ")$"
    app.add_handler(MessageHandler(filters.Regex(names_pattern), claim_task), group=15)

    # Back -> return to main keyboard
    app.add_handler(
        MessageHandler(
            filters.Regex(r"^Back$"),
            lambda u, c: u.message.reply_text("Back to main.", reply_markup=main_keyboard())
        ),
        group=10
    )
