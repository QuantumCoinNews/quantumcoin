"""Public response models for identity, wallet ownership and withdrawal status."""

from typing import Literal

from pydantic import BaseModel


class RouteError(BaseModel):
    ok: Literal[False]
    error: str


class RouteOK(BaseModel):
    ok: Literal[True]


class HealthResponse(RouteOK):
    ts: int


class IdentitySafety(BaseModel):
    readOnly: bool
    changesDb: bool
    changesBalances: bool
    startsMining: bool
    changesChain: bool


class IdentityStatusResponse(RouteOK):
    diagnosticVersion: Literal["TG_WEBAPP_IDENTITY_V1"]
    tokenConfigured: bool
    authRequired: bool
    devFallbackAllowed: bool
    playerIdSource: Literal["server_verified_telegram_session"]
    safeBoundary: IdentitySafety


class TelegramUser(BaseModel):
    id: int | str
    username: str | None
    first_name: str | None
    last_name: str | None
    language_code: str | None


class IdentityVerificationResponse(RouteOK):
    playerId: str
    telegramUser: TelegramUser
    authDate: int
    ageSec: int
    hashOk: Literal[True]
    rawKeys: list[str]
    accessToken: str
    tokenType: Literal["Bearer"]
    expiresAt: int
    safeBoundary: IdentitySafety


class WalletChallenge(BaseModel):
    version: Literal["QC_GAME_WALLET_LINK_V1"]
    networkID: str
    genesisHex: str
    playerId: str
    address: str
    walletOwner: str
    nonce: str
    issuedAt: int
    expiresAt: int


class WalletChallengeResponse(RouteOK):
    challenge: WalletChallenge
    signingBytesHex: str


class WalletLinkResponse(RouteOK):
    playerId: str
    qcAddress: str
    ownershipVerified: Literal[True]


class WithdrawRecord(BaseModel):
    id: str
    userId: str
    asset: str
    amount: str
    amount_u: int
    toAddress: str
    status: str
    txHash: str | None
    createdAt: int
    updatedAt: int


class WithdrawStatusResponse(RouteOK):
    request: WithdrawRecord
