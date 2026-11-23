"""QuantumCoin Telegram Game. Canonical backend: telegram_game.runtime:app."""
__version__ = "0.5.0"


def load_tests(loader, tests, pattern):
    # unittest's package hook avoids importing legacy policy scripts as runtime.
    from isolated_contracts import policy_suite
    return policy_suite()
