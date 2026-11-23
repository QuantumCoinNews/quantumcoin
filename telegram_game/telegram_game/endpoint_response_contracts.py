"""Named wire contracts for the canonical game API.

These describe the existing wire format; they do not filter response fields,
coerce ledger values or turn provider failures into successful responses.
Versioned policy/evidence documents owned by another module remain JSON objects,
explicitly marked as extension documents rather than fabricated local DTOs.
"""
from typing import Literal

from fastapi.routing import APIRoute
from fastapi.datastructures import DefaultPlaceholder
from fastapi.responses import HTMLResponse
from pydantic import BaseModel, ConfigDict, JsonValue, create_model

from .response_contracts import RouteError

Document = dict[str, JsonValue]
NullableText = str | None


class WireModel(BaseModel):
    model_config = ConfigDict(extra="forbid", strict=True)


def model(model_name, **fields):
    """Declare required fields by type; (type, default) declares an optional key."""
    return create_model(model_name, __base__=WireModel, **{
        key: value if isinstance(value, tuple) else (value, ...)
        for key, value in fields.items()
    })


class ErrorDocument(RouteError):
    # Guards and fail-closed adapters add diagnostic context to this envelope.
    model_config = ConfigDict(extra="allow", strict=True)
    __pydantic_extra__: dict[str, JsonValue]


ValidationIssue = model("RedactedValidationIssue", loc=list[str | int], type=str, msg=str)
ValidationFailure = model("RedactedValidationFailure", detail=list[ValidationIssue])


TGWTBalances = model("TGWTBalances", earned_u=int, pending_u=int, verified_u=int,
    withdrawn_u=int, rev=int, reserved_u=int, legacyUnreconciledRequests=int,
    accountingConsistent=bool)
WalletSnapshot = model("GameWalletSnapshot", playerId=str, address=str,
    qc_balance=str, qc_atoms=str, atomsPerQc=str, xpTotal=int,
    legacyNumericReviewRequired=bool, tgwt_balance=int, level=int, xp=int, xp_next=int)


class WalletView(WalletSnapshot):
    linked_qc_address: str
    real_qc_balance: str | None
    real_qc_spendable: str | None
    real_qc_height: int | None
    real_qc_ok: bool
    withdrawableQc: str
    legacyQcSettlementExecutable: Literal[False]
    withdrawableRewardCount: int
    requestedWithdrawQc: str
    requestedWithdrawCount: int
    minWithdrawAmount: float
    latestWithdrawBatch: Document | None
    real_qc: Document
    real_qc_atoms: str | None = None
    real_qc_spendable_atoms: str | None = None


LeaderboardRow = model("LeaderboardRow", rank=int, playerId=str, name=str,
    displayName=str, qc=str, qcAtoms=str, sessions=int, lastSeen=int, lastSeenSecAgo=int)
Settings = model("PlayerSettings", playerId=str, lang=NullableText, theme=NullableText)
MiningRun = model("MiningRunStarted", ok=Literal[True], sessionId=str,
    startedAt=int, expiresAt=int, started=bool, siteId=int, wallet=WalletSnapshot, miningMode=str)
MiningComplete = model("MiningRunCompleted", ok=Literal[True], durationSec=int,
    qcEarned=Literal[0], xpEarned=int, rewardId=str, settlementStatus=str,
    blockHash=str, chainHeight=int, miningMode=str, wallet=WalletSnapshot,
    rewardPaidByChain=bool, tgwtReward=Literal[0])
TGWTState = model("TGWTState", ok=Literal[True], userId=str, rev=int,
    balances=TGWTBalances, ts=int)
TGWTEvent = model("TGWTEvent", id=str, kind=str, amount=str, amount_u=int,
    meta=Document, createdAt=int)
Reservation = model("TGWTReservation", ok=Literal[True], requestId=str, status=str,
    balances=TGWTBalances, onChainPaid=bool, alreadyRecorded=(bool, False),
    gasPayer=(str, "user"), gasAsset=(str, "BNB"), fee=(str, "0"), bscClaimEnabled=(bool, False))
Transition = model("TGWTReservationTransition", ok=Literal[True], status=str,
    alreadyRecorded=(bool, False), onChainPaid=(bool, False), balances=(TGWTBalances | None, None))
Reward = model("VerifiedTaskReward", ok=Literal[True], alreadyCredited=bool,
    rewardId=str, credited_u=int, onChainPaid=bool, balances=(TGWTBalances | None, None),
    rewardTGWT=(str | None, None), poolRemaining_u=(int | None, None),
    status=(str | None, None), gasPayer=(str | None, None), gasAsset=(str | None, None))
NFTCandidate = model("MonthlyNFTCandidate", candidateId=str, userId=str, period=str,
    nftType=str, source=str, activity=Document, score=float, aiReviewStatus=str,
    aiRiskLevel=str, adminStatus=str, adminNote=str, winnerStatus=str,
    mintedStatus=str, policyVersion=str, createdAt=int, updatedAt=int)
NFTCandidateResponse = model("MonthlyNFTCandidateResponse", ok=Literal[True],
    candidate=NFTCandidate | None, policy=(Document | None, None),
    alreadyExists=(bool, False), safeBoundary=(Document | None, None), aiReview=(Document | None, None))

# The node owns these documents. The game forwards them without inventing a
# second balance, fee, consensus, work or security authority.
NodeBridge = Document
MiningJob = model("CanonicalMiningJob", job_id=str, protocol=str, network=str,
    genesis_hash=str, prev_hash=str, height=int, miner=str, header=str,
    nonce_offset=int, nonce_encoding=str, max_nonce=str, difficulty=int, expires=int)


class NodeMiningJob(MiningJob):
    model_config = ConfigDict(extra="allow", strict=True)
    __pydantic_extra__: dict[str, JsonValue]


WorkIssued = model("ClientWorkIssued", ok=Literal[True], job=NodeMiningJob)
WorkAccepted = model("ClientWorkAccepted", ok=Literal[True], receipt=Document,
    alreadyAccepted=(bool, False), rewardPaidByChain=(bool, True), current=(bool | None, None),
    gameQCCredit_u=(Literal[0], 0))
WorkStatus = model("ClientWorkStatus", ok=Literal[True], current=bool,
    jobId=(str | None, None), tip=(Document | None, None))


def ok(name, **fields):
    return model(name, ok=Literal[True], **fields)


SUCCESS = {
    "presence_heartbeat": ok("PresenceHeartbeat", totalPlayers=int, onlineNow=int,
        yourRank=int, yourTotalEarnedQC=str),
    "session_complete": ErrorDocument,
    "social_claim": Reward, "store_buy": Reward, "watch_claim": Reward,
    "verified_reward_claim": Reward,
    "stats_global": model("GlobalGameStats", totalPlayers=int, onlineNow=int,
        yourRank=int, yourTotalEarnedQC=str, totalQCEarned=str),
    "leaderboard_top": model("Leaderboard", rows=list[LeaderboardRow]),
    "wallet": WalletView,
    "store": model("StoreCatalog", ships=list[model("StoreShip", id=str, name=str, price_qc=int)]),
    "social_links": model("SocialLinks", links=dict[str, str]),
    "watch": model("WatchPolicy", enabled=bool, daily_limit=int, reward_tgwt=int),
    "get_settings": Settings,
    "save_settings": ok("SavedSettings", playerId=str, lang=NullableText, theme=NullableText),
    "bridge_status": model("QCBridgeStatus", ok=bool, mode=str, gameMiningMode=str,
        diagnosticVersion=str, playerId=str, qcApiBase=str, configError=str | None,
        qcApiHealth=NodeBridge, linked=bool, targetQcAddress=str, realQc=NodeBridge, safeBoundary=Document),
    "mining_mode_status": ok("GameMiningMode", diagnosticVersion=str, miningMode=str,
        realChainMining=bool, clientPoWRequired=bool, serverMinesForPlayer=Literal[False],
        secondaryGameQCPayment=Literal[False], testnetSimulated=bool, disabled=bool, safeBoundary=Document),
    "mine_start": MiningRun, "mine_finish": MiningComplete,
    "tgwt_state": TGWTState,
    "tgwt_ledger": ok("TGWTLedger", userId=str, pool=Document, rev=int,
        balances=TGWTBalances, events=list[TGWTEvent], ts=int),
    "tgwt_earn": ErrorDocument, "tgwt_verify": ErrorDocument,
    "withdraw_request": Reservation, "withdraw_cancel": Transition, "withdraw_reject": Transition,
    "withdraw_confirm_unavailable": ErrorDocument,
    "withdraw_list": ok("TGWTWithdrawals", items=list[Document]),
    "rewards_history": ok("RewardHistory", playerId=str, rows=list[Document]),
    "qc_node_health": NodeBridge, "qc_address_balance": NodeBridge,
    "qc_tx_status": ok("QCTransactionReceipt", txid=str, source=str, status=str,
        confirmed=bool, height=int, qcNode=NodeBridge),
    "wallet_tx_history": ok("QCTransactionHistory", playerId=str, count=int, items=list[Document]),
    "wallet_real_balance": model("LinkedQCRealBalance", ok=bool, playerId=str,
        linked=bool, qcAddress=str, realBalance=NodeBridge),
    "wallet_send_qc": model("SignedQCBroadcastReceipt", ok=bool, txid=str,
        status=str, qcSendResult=NodeBridge),
    "qc_withdraw_status": ok("LegacyQCWithdrawals", playerId=str, count=int, rows=list[Document]),
    "settlement_summary": ok("LegacySettlementSummary", summary=Document),
    "dev_reset": ErrorDocument,
    "nft_monthly_session_start": ok("NFTSessionStarted", sessionId=str,
        status=Literal["STARTED"], reused=(bool, False)),
    "nft_monthly_session_complete": ok("NFTSessionCompleted", sessionId=str,
        period=str, status=Literal["COMPLETED"], activityVerified=Literal[True]),
    "nft_monthly_candidate_create": NFTCandidateResponse,
    "nft_monthly_candidate_status": ok("NFTCandidateStatus", userId=str, period=str,
        candidate=NFTCandidate | None, policyVersion=str),
    "admin_nft_monthly_candidates": ok("NFTCandidates", period=str, count=int,
        candidates=list[NFTCandidate], policy=Document),
    "admin_nft_monthly_review": NFTCandidateResponse,
    "admin_nft_monthly_ai_review": NFTCandidateResponse,
    "admin_nft_monthly_select_winner": ErrorDocument,
    "nft_wallet_challenge_v1": ok("NFTWalletChallenge", challenge=Document, signingBytesHex=str),
    "nft_wallet_prove_v1": ok("NFTWalletProof", walletOwner=str, period=str),
    "admin_nft_monthly_freeze_pool_v1": ok("NFTFrozenPoolCreated", alreadyFrozen=bool,
        period=str, version=str, poolHash=str, manifest=Document, frozenAt=int),
    "admin_nft_monthly_frozen_pool_v1": ok("NFTFrozenPool", period=str, version=str,
        poolHash=str, manifest=Document, frozenAt=int),
    "admin_nft_draw_commitment_v1": ok("NFTDrawCommitment", commitment=Document, winnerSelected=bool),
    "admin_nft_register_draw_commitment_v1": ok("NFTDrawCommitmentRegistered",
        commitment=Document, alreadyRegistered=bool),
    "admin_nft_publish_draw_v1": ok("NFTDrawCommitmentPublished",
        commitment=Document, alreadyRegistered=bool),
    "admin_nft_verified_draw_v1": ok("NFTVerifiedDraw", receipt=Document,
        receiptHash=str, alreadySelected=bool),
    "admin_nft_draw_from_beacon_v1": ok("NFTBeaconDraw", receipt=Document,
        receiptHash=str, alreadySelected=bool),
    "admin_nft_program_catalog_draft_v1": ok("NFTCatalogDraft", draft=Document, filesWritten=Literal[False]),
    "admin_nft_final_approval_statement_v1": ok("NFTFinalApprovalStatement",
        statement=Document, signingBytesHex=Document),
    "admin_nft_final_readiness_v1": ok("NFTFinalReadiness", readiness=Document),
    "admin_nft_monthly_controlled_mint": ok("NFTMintReceipt", version=str,
        candidateId=str, mintConfirmed=Literal[True], receipt=Document),
    "qc26_admin_panel_registry": ok("AdminPanelRegistry", version=str,
        pagination=Document, totalGroups=int, groups=list[Document], safeBoundary=Document),
    "qc26_admin_panel_pagination_policy": ok("AdminPaginationPolicy", version=str,
        pagination=Document, summary=Document),
    "qc27_admin_monitoring_event_policy": ok("MonitoringEventPolicy", version=str,
        validation=Document, classification=Document, runbook=Document),
    "start_x_reward_verification": ok("XVerificationStarted", authorizeUrl=str,
        state=str, expiresAt=int) | model("XRewardAlreadyCredited", ok=Literal[True],
        alreadyCredited=Literal[True], rewardId=str, credited_u=Literal[0]),
    "complete_x_reward_verification": Reward,
    "verify_telegram_reward": Reward,
    "start_youtube_reward_verification": ok("YouTubeVerificationStarted", authorizeUrl=str,
        state=str, expiresAt=int) | model("YouTubeRewardAlreadyCredited", ok=Literal[True],
        alreadyCredited=Literal[True], rewardId=str, credited_u=Literal[0]),
    "complete_youtube_reward_verification": Reward,
    "create_verified_reward_campaign": model("RewardCampaignCreated", ok=bool,
        campaignHash=str, campaignId=(str | None, None), error=(str | None, None)),
    "verified_reward_pool": ok("TGWTRewardPool", total_u=int, allocated_u=int,
        remaining_u=int, offChainAccounting=Literal[True], onChainFundingVerified=Literal[False],
        gasPayer=Literal["user"], gasAsset=Literal["BNB"], tokenContract=str, bscClaimEnabled=Literal[False],
        socialRewardsEnabled=bool, socialRewardsNetwork=str),
    "verified_reward_campaigns": ok("TGWTRewardCampaigns", campaigns=list[Document],
        verificationRequired=Literal[True], providerAdaptersConfiguredAutomatically=Literal[False],
        socialRewardsEnabled=bool, socialRewardsNetwork=str),
    "verified_reward_campaign_state": model("RewardCampaignState", ok=bool),
    "issue_game_client_work": WorkIssued | WorkAccepted | WorkStatus,
    "game_client_work_status": WorkAccepted | WorkStatus,
    "submit_game_client_work": WorkAccepted,
    "persisted_runtime_limit_status": ok("DurableLimitStatus", backend=str,
        activeRows=int, maxRows=int, actorFrom=str, trustClientForwardedHeaders=Literal[False],
        expiredRowsPrunedOnAdmission=Literal[True]),
}

for handler in ("settlement_queue", "settlement_pending", "settlement_confirmed"):
    SUCCESS[handler] = ok("LegacySettlementRows", status=str, count=int, rows=list[Document])
for handler in ("settlement_mark_pending", "settlement_mock_confirm", "qc_withdraw_request",
                "qc_withdraw_mark_pending", "qc_withdraw_mock_confirm", "qc_withdraw_real_payout_precheck"):
    SUCCESS[handler] = ErrorDocument

SUCCESS.update({
    "_qc_security_action_burst_status_v1": ok("ActionBurstStatus", diagnosticVersion=str,
        enabled=bool, protectedPathCount=int, protectedPaths=list[Document], activeCounters=list[Document],
        recentBlocks=list[Document], summary=Document, safeBoundary=Document),
    "_qc_security_action_burst_test_ping_v1": ok("ActionBurstPing", diagnosticVersion=str,
        testOnly=bool, playerId=str, safeBoundary=Document),
    "_qc_security_rate_limit_status_v1": ok("RateLimitStatus", diagnosticVersion=str,
        sourceSignals=Document, summary=Document, actionRiskRows=list[Document],
        knownGaps=list[JsonValue], runtimeDb=Document, safeBoundary=Document),
    "_qc_security_conflicts_status_v1": ok("ConflictStatus", diagnosticVersion=str,
        tablesSeen=list[str], summary=Document, duplicateQcAddresses=list[Document],
        duplicateRewardGroups=list[Document], duplicateQcTxIds=list[Document],
        withdrawStatusGroups=list[Document], qcTxStatusCounts=list[Document],
        qcTxRecent=list[Document], openWithdraws=list[Document], readyForChainRewards=list[Document],
        idempotencyRecent=list[Document], safeBoundary=Document),
    "_qc_idempotency_status_v1": ok("IdempotencyStatus", guard=str, states=dict[str, int],
        guarantee=str, automaticRetry=Literal[False]),
    "qc27_admin_monitoring_runbook": ok("MonitoringRunbook", version=str, scope=str,
        severities=list[str], entries=list[Document], safeBoundary=Document),
    "qc29_admin_persistence_db_plan": ok("PersistencePlan", version=str, plan=Document, validation=Document),
    "qc30_admin_queue_worker_plan": ok("QueueWorkerPlan", version=str, plan=Document, validation=Document),
    "qc31_admin_backup_restore_runbook": model("BackupRestoreRunbook", ok=bool,
        runbook=Document, validation=Document, source=str, liveRestoreAvailable=Literal[False]),
    "admin_nft_final_approval_statement_v1": ok("NFTFinalApprovalStatement", statement=Document,
        signingBytesHex=dict[str, str], mintAllowed=Literal[False], signed=Literal[False]),
    "admin_nft_final_readiness_v1": ok("NFTFinalReadiness", assetStage=str, approvalStage=str,
        gameMintStage=str, chainConfirmation=str, readyForSignedAward=bool,
        mintAllowed=Literal[False], reasons=list[str]),
    "admin_nft_monthly_freeze_pool_v1": ok("NFTFrozenPoolCreated", alreadyFrozen=bool,
        period=str, version=str, poolHash=str, candidateCount=int, manifest=Document,
        frozenAt=int, safeBoundary=Document),
    "admin_nft_monthly_frozen_pool_v1": ok("NFTFrozenPool", period=str, version=str,
        poolHash=str, candidateCount=int, manifest=Document, frozenAt=int, safeBoundary=Document),
})
for handler in ("qc32_admin_mainnet_before_economy", "qc32c_admin_burn_sink_control",
                "qc32d_admin_mining_node_transition", "qc32e_admin_network_fee_policy",
                "qc32f_admin_founder_vesting_policy", "qc32g_admin_hardfork_policy"):
    SUCCESS[handler] = ok("CanonicalNodePolicySection", version=str, source=str,
        context=Document, policy=Document, authority=str)


class ContractRoute(APIRoute):
    """Publish contracts without changing existing response serialization."""
    def __init__(self, path, endpoint, **kwargs):
        if endpoint.__name__ == "admin_settlement_panel":
            kwargs["response_class"] = HTMLResponse
            super().__init__(path, endpoint, **kwargs)
            return
        if kwargs.get("include_in_schema", True):
            handler = endpoint.__name__
            declared = kwargs.get("response_model")
            if isinstance(declared, DefaultPlaceholder):
                declared = declared.value
            if declared is None:
                if handler not in SUCCESS:
                    raise ValueError("Missing canonical response contract: " + handler)
                responses = dict(kwargs.get("responses") or {})
                responses.setdefault(200, {"model": SUCCESS[handler] | ErrorDocument,
                                           "description": "Existing wire response; inspect ok/error as well as HTTP status."})
                kwargs["responses"] = responses
            responses = dict(kwargs.get("responses") or {})
            for status in (400, 401, 403, 409, 410, 413, 429, 503):
                responses.setdefault(status, {"model": ErrorDocument,
                    "description": "Guard or handler error, when applicable; never implies feature activation."})
            responses.setdefault(422, {"model": ValidationFailure | ErrorDocument,
                "description": "Redacted validation issues; request inputs and secrets are not echoed."})
            kwargs["responses"] = responses
        super().__init__(path, endpoint, **kwargs)


def install_openapi_contracts(app):
    """Describe the existing outer ASGI authentication boundary explicitly."""
    original = app.openapi

    def openapi():
        from .player_auth import PUBLIC, operator_path
        spec = original()
        spec.setdefault('components', {}).setdefault('securitySchemes', {}).update({
            'TelegramSession': {'type':'http','scheme':'bearer',
                'description':'Server-issued Telegram session; never caller-supplied playerId.'},
            'GameOperatorHeader': {'type':'apiKey','in':'header','name':'X-Admin-Token'},
            'GameOperatorAlternateHeader': {'type':'apiKey','in':'header','name':'X-QC-ADMIN-TOKEN'},
            'GameOperatorBearer': {'type':'http','scheme':'bearer',
                'description':'Separate configured operator credential, not a player session.'},
        })
        for path, entry in spec['paths'].items():
            for method, operation in entry.items():
                if method.upper() not in {'GET','POST','PUT','DELETE','PATCH','HEAD','OPTIONS'}:continue
                if operator_path(path):
                    operation['security']=[{'GameOperatorHeader':[]},{'GameOperatorAlternateHeader':[]},{'GameOperatorBearer':[]}]
                elif path.startswith('/api/') and (method.upper(),path) not in PUBLIC:
                    operation['security']=[{'TelegramSession':[]}]
                else:operation['security']=[]
                if path in {'/api/v1/tgwt/earn','/api/v1/tgwt/verify','/api/dev/reset',
                            '/api/settlement/mark-pending','/api/settlement/mock-confirm',
                            '/api/withdraw/request','/api/withdraw/mark-pending',
                            '/api/withdraw/mock-confirm','/api/withdraw/real-payout-precheck',
                            '/api/admin/tgwt/withdraw/confirm'}:
                    # These handlers always return an explicit error Response.
                    # Do not advertise a fictional HTTP-200 successful feature.
                    operation['responses'].pop('200',None)
        return spec
    app.openapi = openapi
