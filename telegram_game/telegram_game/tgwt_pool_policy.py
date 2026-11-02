# TGWT_POOL_POLICY_V1
#
# This module defines the fixed TGWT policy number only.
# It does NOT create a live pool, database table, payout flow, BSC transfer,
# mint function, withdraw flow, or reward distribution engine.
#
# Current project decision:
# - TGWT maximum pool target: 3,000,000 TGWT
# - Unit: micro-TGWT, 1 TGWT = 1_000_000 units
# - Status: POLICY_DEFINED_POOL_NOT_CREATED
# - Off-chain only until a later approved implementation step

TGWT_POOL_POLICY_VERSION = "TGWT_POOL_POLICY_V1"
TGWT_UNIT = 1_000_000
TGWT_POOL_TOTAL = 3_000_000
TGWT_POOL_TOTAL_U = TGWT_POOL_TOTAL * TGWT_UNIT
TGWT_POOL_STATUS = "POLICY_DEFINED_POOL_NOT_CREATED"


def tgwt_pool_policy() -> dict:
    return {
        "version": TGWT_POOL_POLICY_VERSION,
        "status": TGWT_POOL_STATUS,
        "total": str(TGWT_POOL_TOTAL),
        "total_u": TGWT_POOL_TOTAL_U,
        "unit": TGWT_UNIT,
        "asset": "TGWT",
        "offChainOnly": True,
        "poolCreated": False,
        "runtimeDistributionEnabled": False,
        "ledgerCreated": False,
        "withdrawEnabled": False,
        "bscPayoutEnabled": False,
        "realPayoutEnabled": False,
        "mintEnabled": False,
        "chainWriteEnabled": False,
        "policyNote": (
            "TGWT 3,000,000 is only a fixed policy cap at this stage. "
            "No live pool, no payout, no BSC transfer, no minting, no chain write, "
            "and no user balance mutation is performed by this policy module."
        ),
        "safeBoundary": {
            "policyOnly": True,
            "createsPool": False,
            "createsLedger": False,
            "distributesRewards": False,
            "changesTGWTBalances": False,
            "changesQCBalances": False,
            "performsTransfer": False,
            "mintsToken": False,
            "burnsToken": False,
            "startsWithdraw": False,
            "approvesWithdraw": False,
            "bscPayout": False,
            "realPayout": False,
            "changesChain": False,
            "replacesChain": False,
            "changesConsensus": False,
            "changesTotalSupply": False,
            "startsMining": False,
            "stopsMining": False,
        },
    }


def tgwt_policy_cap_ok(amount_u: int, already_reserved_u: int = 0) -> bool:
    """Pure cap check only. Does not reserve, spend, write DB, or mutate balances."""
    try:
        amount_u = int(amount_u)
        already_reserved_u = int(already_reserved_u)
    except Exception:
        return False

    if amount_u <= 0:
        return False
    if already_reserved_u < 0:
        return False

    return already_reserved_u + amount_u <= TGWT_POOL_TOTAL_U


def tgwt_policy_remaining_u(already_reserved_u: int = 0) -> int:
    """Pure remaining calculation only. Does not create a live pool."""
    try:
        already_reserved_u = int(already_reserved_u)
    except Exception:
        return 0

    if already_reserved_u < 0:
        return 0

    remaining = TGWT_POOL_TOTAL_U - already_reserved_u
    if remaining < 0:
        return 0
    return remaining
