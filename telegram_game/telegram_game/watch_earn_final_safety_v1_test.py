# WATCH_EARN_TESTNET_FINAL_LOCK_V1_TEST

import os
import sys
from pathlib import Path
from contextlib import contextmanager

_THIS_FILE = Path(__file__).resolve()
_TG_ROOT = _THIS_FILE.parents[1]
if str(_TG_ROOT) not in sys.path:
    sys.path.insert(0, str(_TG_ROOT))

from telegram_game.watch_earn_final_safety_v1 import (
    WATCH_EARN_FINAL_LOCK_VERSION,
    WATCH_FINAL_APPROVAL,
    WATCH_MODE_DEV,
    WATCH_MODE_FINAL,
    WATCH_MODE_TESTNET,
    WATCH_PROVIDER_ADMOB_REWARDED,
    WATCH_PROVIDER_GOOGLE_REWARDED_ADS,
    WatchEarnFinalSafetyError,
    watch_earn_assert_policy_safe_v1,
    watch_earn_core_todo_v1,
    watch_earn_frontend_public_payload_v1,
    watch_earn_public_policy_from_env_v1,
    watch_earn_resolve_mode_v1,
    watch_earn_validate_completion_event_v1,
    watch_earn_validate_policy_v1,
)


@contextmanager
def clean_env():
    keys = [
        "WATCH_EARN_MODE",
        "WATCH_EARN_NETWORK_MODE",
        "WATCH_EARN_FINAL_ENABLED",
        "WATCH_EARN_PROVIDER",
        "WATCH_EARN_SERVER_SIDE_VERIFICATION_REQUIRED",
        "WATCH_EARN_REWARD_ASSET",
        "WATCH_EARN_REWARD_LEDGER_STATE",
        "WATCH_EARN_FINAL_APPROVAL",
        "QC_TESTNET",
        "QC_CHAIN_MODE",
        "QC_NETWORK",
        "TELEGRAM_GAME_CHAIN_MODE",
        "TELEGRAM_GAME_NETWORK",
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
    except WatchEarnFinalSafetyError:
        return
    raise AssertionError("expected WatchEarnFinalSafetyError: " + label)


def test_default_dev_safe():
    with clean_env():
        policy = watch_earn_public_policy_from_env_v1()
        assert policy["version"] == WATCH_EARN_FINAL_LOCK_VERSION
        assert policy["mode"] == WATCH_MODE_DEV
        assert policy["rewardAsset"] == "TGWT"
        assert policy["rewardLedgerState"] == "pending"
        assert watch_earn_assert_policy_safe_v1(policy)["mode"] == WATCH_MODE_DEV


def test_project_testnet_resolves_testnet():
    with clean_env():
        os.environ["QC_TESTNET"] = "1"
        assert watch_earn_resolve_mode_v1() == WATCH_MODE_TESTNET


def test_final_requires_manual_approval():
    with clean_env():
        policy = {
            "mode": WATCH_MODE_FINAL,
            "finalEnabled": True,
            "provider": WATCH_PROVIDER_GOOGLE_REWARDED_ADS,
            "serverSideVerificationRequired": True,
            "rewardAsset": "TGWT",
            "rewardLedgerState": "pending",
            "revenueLogSeparateFromRewardLog": True,
        }
        valid, reason = watch_earn_validate_policy_v1(policy)
        assert valid is False
        assert reason == "watch_final_requires_explicit_manual_approval"


def test_final_requires_rewarded_provider_and_ssv():
    with clean_env():
        base = {
            "mode": WATCH_MODE_FINAL,
            "finalEnabled": True,
            "provider": "none",
            "serverSideVerificationRequired": True,
            "rewardAsset": "TGWT",
            "rewardLedgerState": "pending",
            "revenueLogSeparateFromRewardLog": True,
            "finalApproval": WATCH_FINAL_APPROVAL,
        }
        valid, reason = watch_earn_validate_policy_v1(base)
        assert valid is False
        assert reason == "watch_final_requires_rewarded_ads_provider"

        base["provider"] = WATCH_PROVIDER_ADMOB_REWARDED
        base["serverSideVerificationRequired"] = False
        valid, reason = watch_earn_validate_policy_v1(base)
        assert valid is False
        assert reason == "watch_final_requires_server_side_verification"


def test_final_valid_policy_and_payload():
    with clean_env():
        policy = {
            "mode": WATCH_MODE_FINAL,
            "finalEnabled": True,
            "provider": WATCH_PROVIDER_GOOGLE_REWARDED_ADS,
            "serverSideVerificationRequired": True,
            "rewardAsset": "TGWT",
            "rewardLedgerState": "pending",
            "revenueLogSeparateFromRewardLog": True,
            "finalApproval": WATCH_FINAL_APPROVAL,
        }
        safe = watch_earn_assert_policy_safe_v1(policy)
        assert safe["mode"] == WATCH_MODE_FINAL

        payload = watch_earn_frontend_public_payload_v1(policy)
        assert payload["finalMode"] == "GOOGLE_REWARDED_ADS_VERIFIED_COMPLETION"
        assert payload["rewardAsset"] == "TGWT"
        assert payload["rewardLedgerState"] == "pending"


def test_final_completion_requires_verified_rewarded_ssv():
    with clean_env():
        policy = {
            "mode": WATCH_MODE_FINAL,
            "finalEnabled": True,
            "provider": WATCH_PROVIDER_GOOGLE_REWARDED_ADS,
            "serverSideVerificationRequired": True,
            "rewardAsset": "TGWT",
            "rewardLedgerState": "pending",
            "revenueLogSeparateFromRewardLog": True,
            "finalApproval": WATCH_FINAL_APPROVAL,
        }

        ok, reason = watch_earn_validate_completion_event_v1({
            "completionType": "rewarded_completed",
            "verifiedRewardedCompletion": True,
            "serverSideVerificationValid": True,
            "devSimulation": False,
        }, policy)
        assert ok is True
        assert reason == "ok"

        ok, reason = watch_earn_validate_completion_event_v1({
            "completionType": "ad_loaded",
            "verifiedRewardedCompletion": True,
            "serverSideVerificationValid": True,
            "devSimulation": False,
        }, policy)
        assert ok is False
        assert reason == "watch_final_rejects_non_rewarded_completion_type"

        ok, reason = watch_earn_validate_completion_event_v1({
            "completionType": "rewarded_completed",
            "verifiedRewardedCompletion": True,
            "serverSideVerificationValid": False,
            "devSimulation": False,
        }, policy)
        assert ok is False
        assert reason == "watch_final_requires_valid_ssv_event"

        ok, reason = watch_earn_validate_completion_event_v1({
            "completionType": "rewarded_completed",
            "verifiedRewardedCompletion": True,
            "serverSideVerificationValid": True,
            "devSimulation": True,
        }, policy)
        assert ok is False
        assert reason == "watch_final_rejects_dev_simulation"


def test_dev_rejects_load_open_click_rewards():
    with clean_env():
        policy = watch_earn_public_policy_from_env_v1()
        ok, reason = watch_earn_validate_completion_event_v1({
            "completionType": "clicked",
            "verifiedRewardedCompletion": False,
            "serverSideVerificationValid": False,
            "devSimulation": True,
        }, policy)
        assert ok is False
        assert reason == "watch_reward_not_allowed_for_load_open_or_click"


def test_todo_contains_final_steps():
    todo = watch_earn_core_todo_v1()
    assert "wire_google_rewarded_ads_sdk_later" in todo
    assert "wire_server_side_verification_later" in todo
    assert "keep_tgwt_reward_pending_ledger_only" in todo
    assert "keep_mainnet_activation_manual_approval" in todo


if __name__ == "__main__":
    test_default_dev_safe()
    test_project_testnet_resolves_testnet()
    test_final_requires_manual_approval()
    test_final_requires_rewarded_provider_and_ssv()
    test_final_valid_policy_and_payload()
    test_final_completion_requires_verified_rewarded_ssv()
    test_dev_rejects_load_open_click_rewards()
    test_todo_contains_final_steps()
    print("WATCH_EARN_TESTNET_FINAL_LOCK_V1_TEST_PASS")
