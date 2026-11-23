"""Retired independent ledger. Existing legacy DB files are preserved for audit.

No automatic conversion of unverified balances or synthetic confirmations is safe.
The canonical connection factory is telegram_game.dev_api._db, used by runtime.
"""
class Store:
    def __init__(self, *args, **kwargs):
        raise RuntimeError("Independent Telegram Store retired; use telegram_game.runtime")
