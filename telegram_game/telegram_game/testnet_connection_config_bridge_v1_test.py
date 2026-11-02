# QC_TG_TESTNET_CONNECTION_CONFIG_BRIDGE_V1_TEST

import os
import sys
from pathlib import Path
from contextlib import contextmanager

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game import config as tg_config
from telegram_game.testnet_connection_v1 import (
    DEFAULT_LOCAL_QC_API_BASE,
    DEFAULT_TESTNET_QC_API_BASE,
    TelegramGameConnectionError,
)


@contextmanager
def clean_env():
    keys = [
        "TELEGRAM_GAME_CHAIN_MODE",
        "TELEGRAM_GAME_NETWORK",
        "QC_CHAIN_MODE",
        "QC_NETWORK",
        "QC_MODE",
        "QC_NET_MAGIC",
        "QC_TESTNET",
        "QC_PUBLIC_TESTNET",
        "QC_API_BASE",
        "QUANTUMCOIN_API_BASE",
        "QC_NODE_API",
        "TELEGRAM_GAME_ALLOW_TESTNET_MAIN_PORT",
    ]
    old = {k: os.environ.get(k) for k in keys}
    try:
        for k in keys:
            os.environ.pop(k, None)
        yield
    finally:
        for k, v in old.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v


def expect_error(fn, label):
    try:
        fn()
    except TelegramGameConnectionError:
        return
    raise AssertionError("expected TelegramGameConnectionError: " + label)


def test_config_get_api_base_local_default():
    with clean_env():
        assert tg_config.get_api_base() == DEFAULT_LOCAL_QC_API_BASE


def test_config_get_api_base_testnet_default():
    with clean_env():
        os.environ["QC_TESTNET"] = "1"
        assert tg_config.get_api_base() == DEFAULT_TESTNET_QC_API_BASE


def test_config_get_api_base_testnet_rejects_8082():
    with clean_env():
        os.environ["QC_TESTNET"] = "1"
        os.environ["QC_API_BASE"] = "http://127.0.0.1:8082"
        expect_error(tg_config.get_api_base, "config bridge must reject testnet 8082")


def test_config_get_connection_info():
    with clean_env():
        os.environ["QC_CHAIN_MODE"] = "testnet"
        info = tg_config.get_connection_info_v1()
        assert info["version"] == "QC_TG_TESTNET_CONNECTION_V1"
        assert info["isTestnet"] is True
        assert info["qcApiBase"] == DEFAULT_TESTNET_QC_API_BASE


if __name__ == "__main__":
    test_config_get_api_base_local_default()
    test_config_get_api_base_testnet_default()
    test_config_get_api_base_testnet_rejects_8082()
    test_config_get_connection_info()
    print("QC_TG_TESTNET_CONNECTION_CONFIG_BRIDGE_V1_TEST_PASS")
