# QC_TG_TESTNET_CONNECTION_V1_TEST

import os
import sys
from pathlib import Path
from contextlib import contextmanager

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.testnet_connection_v1 import (
    DEFAULT_LOCAL_QC_API_BASE,
    DEFAULT_TESTNET_QC_API_BASE,
    NETWORK_LABEL_PATH,
    TelegramGameConnectionError,
    assert_telegram_game_connection_safe_v1,
    get_qc_api_base_for_mode_v1,
    telegram_game_chain_mode_v1,
    telegram_game_connection_info_v1,
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


def test_default_local_base():
    with clean_env():
        assert telegram_game_chain_mode_v1() == "local"
        assert get_qc_api_base_for_mode_v1() == DEFAULT_LOCAL_QC_API_BASE


def test_testnet_default_base():
    with clean_env():
        os.environ["QC_TESTNET"] = "1"
        assert telegram_game_chain_mode_v1() == "testnet"
        assert get_qc_api_base_for_mode_v1() == DEFAULT_TESTNET_QC_API_BASE


def test_testnet_rejects_main_port():
    with clean_env():
        os.environ["QC_TESTNET"] = "1"
        os.environ["QC_API_BASE"] = "http://127.0.0.1:8082"
        expect_error(get_qc_api_base_for_mode_v1, "testnet must reject 8082")


def test_local_rejects_testnet_port():
    with clean_env():
        os.environ["QC_API_BASE"] = "http://127.0.0.1:18082"
        expect_error(get_qc_api_base_for_mode_v1, "local must reject 18082")


def test_testnet_info_flags():
    with clean_env():
        os.environ["QC_CHAIN_MODE"] = "testnet"
        info = telegram_game_connection_info_v1()
        assert info["version"] == "QC_TG_TESTNET_CONNECTION_V1"
        assert info["isTestnet"] is True
        assert info["networkLabelPath"] == NETWORK_LABEL_PATH
        assert info["testnetSimulatedMiningOnly"] is True
        assert info["realChainMiningAllowed"] is False


def test_assert_safe_mainnet_local():
    with clean_env():
        info = assert_telegram_game_connection_safe_v1()
        assert info["isTestnet"] is False
        assert info["qcApiBase"] == DEFAULT_LOCAL_QC_API_BASE


if __name__ == "__main__":
    test_default_local_base()
    test_testnet_default_base()
    test_testnet_rejects_main_port()
    test_local_rejects_testnet_port()
    test_testnet_info_flags()
    test_assert_safe_mainnet_local()
    print("QC_TG_TESTNET_CONNECTION_V1_TEST_PASS")
