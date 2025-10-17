from telegram.ext import CommandHandler
from handlers.core import cmd_start  # absolute import

def wire(app):
    app.add_handler(CommandHandler("start", cmd_start), group=-90)
