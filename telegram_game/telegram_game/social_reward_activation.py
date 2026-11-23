"""Fail-closed launch gate for TGWT social rewards (BSC mainnet only)."""
from __future__ import annotations

import json
import os
from contextlib import closing

from .tgwt_bsc_claim_safety_v1 import (
    TGWT_BSC_NETWORK_MAINNET,
    tgwt_bsc_public_config_from_env_v1,
    tgwt_validate_bsc_public_config_v1,
)


def social_rewards_enabled() -> bool:
    """Keep social TGWT campaigns off until explicitly enabled on BSC mainnet."""
    if (os.getenv("QC_TGWT_SOCIAL_REWARDS_ENABLED", "").strip().lower()
            not in {"1", "true", "yes", "on", "enabled"}):
        return False
    project_mode = " ".join((
        os.getenv("QC_TESTNET", ""), os.getenv("QC_CHAIN_MODE", ""),
        os.getenv("QC_NETWORK", ""), os.getenv("TELEGRAM_GAME_CHAIN_MODE", ""),
        os.getenv("TELEGRAM_GAME_NETWORK", ""),
    )).lower()
    if os.getenv("QC_TESTNET", "").strip().lower() in {"1", "true", "yes", "on", "enabled"} or "testnet" in project_mode:
        return False
    try:
        config = tgwt_bsc_public_config_from_env_v1()
        valid, _ = tgwt_validate_bsc_public_config_v1(config)
        return bool(valid and config["network"] == TGWT_BSC_NETWORK_MAINNET
                    and config["chainId"] == 56 and config["claimEnabled"] is True)
    except Exception:
        return False


def is_social_campaign(db, campaign_id: str) -> bool:
    try:
        with closing(db()) as conn:
            row = conn.execute("SELECT policy_json FROM tgwt_campaigns_v1 WHERE campaign_id=?", (campaign_id,)).fetchone()
        if not row:
            return False
        policy = json.loads(row[0])
        return str(policy.get("provider", "")).upper() in {"X", "TELEGRAM", "YOUTUBE"} and str(policy.get("action", "")).startswith("SOCIAL_")
    except Exception:
        # Invalid campaign state is handled by the reward verifier, fail closed here.
        return True
