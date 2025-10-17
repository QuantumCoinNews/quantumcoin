from telegram.ext import MessageHandler, CommandHandler, filters
from handlers.social import claim_tgwt_btn  # absolute import

def wire(app):
    app.add_handler(MessageHandler(filters.Regex(r"^Claim$"), claim_tgwt_btn), group=10)
    app.add_handler(CommandHandler("claim", claim_tgwt_btn), group=10)
