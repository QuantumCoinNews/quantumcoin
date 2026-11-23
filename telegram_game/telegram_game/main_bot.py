"""Messaging adapter only. Balances, missions and withdrawals belong to runtime.

This module does not maintain a second game economy or confirm withdrawals.
"""
import os
from urllib.parse import urlparse


def build_application():
    from telegram import InlineKeyboardButton, InlineKeyboardMarkup, WebAppInfo
    from telegram.ext import Application, CommandHandler
    token = (os.getenv("TELEGRAM_BOT_TOKEN") or os.getenv("QC_TELEGRAM_BOT_TOKEN") or "").strip()
    url = (os.getenv("QC_GAME_WEBAPP_URL") or "").strip()
    parsed = urlparse(url)
    if not token or parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password:
        raise ValueError("Configure TELEGRAM_BOT_TOKEN and an HTTPS QC_GAME_WEBAPP_URL for the canonical /game")

    async def start(update, context):
        if update.effective_message:
            await update.effective_message.reply_text(
                "Open QuantumCoin Game to view your account and play.",
                reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("Open Game", web_app=WebAppInfo(url=url))]]))

    app = Application.builder().token(token).build()
    app.add_handler(CommandHandler("start", start))
    return app


if __name__ == "__main__":
    build_application().run_polling()
