# handlers/mining.py — ASCII-only
from telegram.ext import CommandHandler
from handlers.moon import open_moon  # /mine -> open moon menu

def wire(app):
    app.add_handler(CommandHandler("mine", open_moon), group=10)
