# main_bot.py — PTB 20.3, ASCII-only (final)
import logging
import importlib

from database.sqlite_store import init_db

from telegram.ext import (
    Application, ApplicationBuilder, MessageHandler, CommandHandler, filters
)

from config import get_token
from handlers import core as core_handlers
from handlers import moon as moon_handlers
from handlers import social as social_handlers
from handlers.core import main_keyboard  # global Back handler için
from utils.state import get_user, flood_guard

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")


def emergency_start_hook(app: Application) -> None:
    # Highest priority /start so nothing can block it
    app.add_handler(CommandHandler("start", core_handlers.cmd_start), group=-100)

    # Safety aliases (typos)
    async def _s(u, c):
        try:
            await core_handlers.cmd_start(u, c)
        except Exception:
            await u.message.reply_text("Start is temporarily unavailable, try /start again.")
    app.add_handler(CommandHandler(["Start", "satart", "staart"], _s), group=-200)

    # Global Back -> her durumda ana menü
    app.add_handler(
        MessageHandler(
            filters.Regex(r"^Back$"),
            lambda u, c: u.message.reply_text("Back to main.", reply_markup=main_keyboard())
        ),
        group=-50
    )


def _fallback_referral(app: Application):
    async def ref_btn(u, c):
        usr = get_user(u)
        if not flood_guard(usr):
            return
        await u.message.reply_text("Referral: coming soon.")
    app.add_handler(MessageHandler(filters.Regex(r"^Referral$"), ref_btn))
    app.add_handler(CommandHandler("referral", ref_btn))


def _fallback_leaderboard(app: Application):
    async def lb_btn(u, c):
        usr = get_user(u)
        if not flood_guard(usr):
            return
        await u.message.reply_text("Leaderboard: coming soon.")
    app.add_handler(MessageHandler(filters.Regex(r"^Leaderboard$"), lb_btn))
    app.add_handler(CommandHandler("leaderboard", lb_btn))


def _fallback_wallet(app: Application):
    async def wallet_btn(u, c):
        usr = get_user(u)
        if not flood_guard(usr):
            return
        await u.message.reply_text("Wallet: not linked yet.")
    app.add_handler(MessageHandler(filters.Regex(r"^Wallet$"), wallet_btn))
    app.add_handler(CommandHandler("wallet", wallet_btn))


def _fallback_profile(app: Application):
    async def profile_btn(u, c):
        usr = get_user(u)
        if not flood_guard(usr):
            return
        await u.message.reply_text(
            f"Profile\nQC: {usr.qc}\nTGWT: {usr.tgwt}\nEnergy: {usr.energy}/100\nLevel: {usr.level}"
        )
    app.add_handler(MessageHandler(filters.Regex(r"^Profile$"), profile_btn))
    app.add_handler(CommandHandler("profile", profile_btn))


def _fallback_claim(app: Application):
    async def claim_btn(u, c):
        usr = get_user(u)
        if not flood_guard(usr):
            return
        await u.message.reply_text("Use Social -> Claim TGWT for social rewards.")
    app.add_handler(MessageHandler(filters.Regex(r"^Claim$"), claim_btn))
    app.add_handler(CommandHandler("claim", claim_btn))


OPTIONALS = {
    "referral": _fallback_referral,
    "leaderboard": _fallback_leaderboard,
    "wallet": _fallback_wallet,
    "profile": _fallback_profile,
    "claim": _fallback_claim,
    "mining": None,   # if module has wire(app) it will be used
    "start":  None,   # core already provides /start
    "shop":   None
}


def _wire_optional_modules(app: Application):
    for name, fallback in OPTIONALS.items():
        try:
            mod = importlib.import_module(f"handlers.{name}")
            if hasattr(mod, "wire"):
                mod.wire(app)
            elif fallback:
                fallback(app)
        except Exception as e:
            logging.warning("Optional module '%s' failed to load: %s", name, e)
            if fallback:
                fallback(app)


def build_app() -> Application:
    app = ApplicationBuilder().token(get_token()).build()

    # 1) make sure /start cannot be blocked
    emergency_start_hook(app)

    # 2) core commands and main buttons
    core_handlers.wire_commands(app)

    # 3) QC mining (moon)
    moon_handlers.wire(app)

    # 4) TGWT socials
    social_handlers.wire(app)

    # 5) other modules (wire if present; else safe fallback)
    _wire_optional_modules(app)

    return app


def main() -> None:
    init_db()  # idempotent
    app = build_app()

    async def _startup(_: Application):
        try:
            me = await app.bot.get_me()
            logging.info("Bot up: @%s id=%s", me.username, me.id)
            await app.bot.delete_webhook(drop_pending_updates=True)
        except Exception as e:
            logging.exception("Startup error: %s", e)

    app.post_init = _startup
    app.run_polling()


if __name__ == "__main__":
    main()
