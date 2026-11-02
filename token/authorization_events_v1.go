package token

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"quantumcoin/security"
)

const (
	TokenAuthorizationAcceptedEventV1       = "TOKEN_AUTHORIZATION_ACCEPTED"
	TokenAuthorizationRejectedEventV1       = "TOKEN_AUTHORIZATION_REJECTED"
	TokenAuthorizationReplayRejectedEventV1 = "TOKEN_AUTHORIZATION_REPLAY_REJECTED"
	TokenOperationRejectedEventV1           = "TOKEN_OPERATION_REJECTED"
	TokenRegistryLoadAcceptedEventV1        = "TOKEN_REGISTRY_LOAD_ACCEPTED"
	TokenRegistryLoadRejectedEventV1        = "TOKEN_REGISTRY_LOAD_REJECTED"
	TokenFungiblePersistenceAcceptedEventV1 = "TOKEN_FUNGIBLE_PERSISTENCE_ACCEPTED"
	TokenFungiblePersistenceRejectedEventV1 = "TOKEN_FUNGIBLE_PERSISTENCE_REJECTED"
	TokenMinerRewardPendingEventV1          = "TOKEN_MINER_REWARD_PENDING"
	TokenMinerRewardFinalizedEventV1        = "TOKEN_MINER_REWARD_FINALIZED"
	TokenMinerRewardDuplicateEventV1        = "TOKEN_MINER_REWARD_DUPLICATE"
	TokenMinerRewardReorgRejectedEventV1    = "TOKEN_MINER_REWARD_REORG_REJECTED"
	TokenMinerRewardRejectedEventV1         = "TOKEN_MINER_REWARD_REJECTED"
	TokenMinerRewardPersistenceEventV1      = "TOKEN_MINER_REWARD_PERSISTENCE_REJECTED"
)

const (
	tokenAuthorizationStageAuthorizationV1 = "authorization"
	tokenAuthorizationStageOperationV1     = "operation"
)

func tokenAuthorizationEventDirectoryV1() string {
	if directory := strings.TrimSpace(
		os.Getenv("QC_NODE_DIR"),
	); directory != "" {
		return directory
	}

	return "."
}

func tokenAuthorizationErrorClassV1(
	err error,
) string {
	switch {
	case err == nil:
		return "NONE"
	case errors.Is(err, ErrTokenAuthorizationReplay):
		return "TOKEN_AUTHORIZATION_REPLAY"
	case errors.Is(err, ErrTokenAuthorizationNonce):
		return "TOKEN_AUTHORIZATION_NONCE"
	case errors.Is(err, ErrTokenAuthorizationRequired):
		return "TOKEN_AUTHORIZATION_REQUIRED"
	case errors.Is(err, ErrTokenAuthorizationInvalid):
		return "TOKEN_AUTHORIZATION_INVALID"
	case errors.Is(
		err,
		ErrQC20FixedSupplyMintDisabled,
	):
		return "QC20_FIXED_SUPPLY_MINT_DISABLED"
	default:
		return "TOKEN_OPERATION_ERROR"
	}
}

// observeFungiblePersistenceV1 records model-aware advisory metadata only.
// It cannot authorize decoding or mutate token, registry, wallet, chain,
// mining, consensus, or persistence state.
func observeFungiblePersistenceV1(
	model FungibleTokenModelV1,
	outcome string,
	loadErr error,
) {
	eventType := TokenFungiblePersistenceAcceptedEventV1
	severity := security.SeverityInfo
	message := "Fungible token persistence accepted."

	if loadErr != nil {
		eventType = TokenFungiblePersistenceRejectedEventV1
		severity = security.SeverityHigh
		message = "Fungible token persistence rejected."
	}

	errorClass := "NONE"

	if loadErr != nil {
		switch {
		case errors.Is(
			loadErr,
			ErrFungiblePersistenceVersionV1,
		):
			errorClass = "FUNGIBLE_PERSISTENCE_VERSION"

		case errors.Is(
			loadErr,
			ErrFungiblePersistenceModelV1,
		):
			errorClass = "FUNGIBLE_PERSISTENCE_MODEL"

		case errors.Is(
			loadErr,
			ErrQC20PersistenceIntegrityV1,
		):
			errorClass = "QC20_PERSISTENCE_INTEGRITY"

		case errors.Is(
			loadErr,
			ErrFungiblePersistenceIntegrityV1,
		):
			errorClass = "FUNGIBLE_PERSISTENCE_INTEGRITY"

		default:
			errorClass = "FUNGIBLE_PERSISTENCE_ENVELOPE"
		}
	}

	normalizedModel := strings.TrimSpace(
		string(model),
	)

	if normalizedModel == "" {
		normalizedModel = "UNKNOWN"
	}

	normalizedOutcome := strings.TrimSpace(outcome)
	if normalizedOutcome == "" {
		normalizedOutcome = "UNKNOWN"
	}

	metadata := map[string]any{
		"model":               normalizedModel,
		"operation":           TokenOperationFungiblePersistenceV1,
		"outcome":             normalizedOutcome,
		"errorClass":          errorClass,
		"aiReadOnly":          true,
		"advisoryOnly":        true,
		"authorizesOperation": false,
		"changesTokenState":   false,
		"changesRegistry":     false,
		"changesChain":        false,
		"changesConsensus":    false,
	}

	event := security.NewEvent(
		eventType,
		severity,
		"token",
		"fungible_persistence",
		message,
		metadata,
	)

	// Observation must never change the decode result.
	_ = security.AppendEvent(
		tokenAuthorizationEventDirectoryV1(),
		event,
	)
}

// observeTokenRegistryLoadV1 writes advisory security metadata only.
// It never authorizes a load, changes registry state, or exposes registry
// contents, wallet material, signatures, or private keys.
func observeTokenRegistryLoadV1(
	directory string,
	outcome string,
	tokenCount int,
	loadErr error,
) {
	eventType := TokenRegistryLoadAcceptedEventV1
	severity := security.SeverityInfo
	message := "Token registry load accepted."

	if loadErr != nil {
		eventType = TokenRegistryLoadRejectedEventV1
		severity = security.SeverityHigh
		message = "Token registry load rejected."
	}

	errorClass := "NONE"
	if loadErr != nil {
		if errors.Is(loadErr, ErrRegistryLoadIntegrityV1) {
			errorClass = "TOKEN_REGISTRY_INTEGRITY"
		} else {
			errorClass = "TOKEN_REGISTRY_LOAD_ERROR"
		}
	}

	normalizedOutcome := strings.TrimSpace(outcome)
	if normalizedOutcome == "" {
		normalizedOutcome = "UNKNOWN"
	}

	metadata := map[string]any{
		"model":               TokenAuthorizationModelLegacyV1,
		"operation":           TokenOperationRegistryLoadV1,
		"outcome":             normalizedOutcome,
		"tokenCount":          tokenCount,
		"errorClass":          errorClass,
		"aiReadOnly":          true,
		"advisoryOnly":        true,
		"authorizesOperation": false,
		"changesRegistry":     false,
	}

	event := security.NewEvent(
		eventType,
		severity,
		"token",
		"token_registry",
		message,
		metadata,
	)

	eventDirectory := strings.TrimSpace(directory)
	if eventDirectory == "" {
		eventDirectory = tokenAuthorizationEventDirectoryV1()
	}

	// Observation must never change the Registry.Load result.
	_ = security.AppendEvent(eventDirectory, event)
}

// observeMinerTokenRewardV1 emits read-only advisory metadata for the
// miner-token bridge. It cannot authorize rewards or change token, registry,
// blockchain, wallet, mining, or consensus state.
func observeMinerTokenRewardV1(
	outcome string,
	symbol string,
	height int,
	blockHash string,
	minerAddress string,
	reward uint64,
	confirmationDepth int,
	operationErr error,
) {
	eventType := TokenMinerRewardPendingEventV1
	severity := security.SeverityInfo
	message := "Miner-token reward is pending finality."

	switch outcome {
	case "FINALIZED":
		eventType = TokenMinerRewardFinalizedEventV1
		message = "Miner-token reward finalized."
	case "DUPLICATE_PENDING", "DUPLICATE_COMMITTED":
		eventType = TokenMinerRewardDuplicateEventV1
		message = "Duplicate miner-token callback ignored."
	case "REORG_REJECTED", "PENDING_REORG_REPLACED":
		eventType = TokenMinerRewardReorgRejectedEventV1
		severity = security.SeverityHigh
		message = "Non-canonical miner-token reward rejected."
	case "PERSISTENCE_REJECTED":
		eventType = TokenMinerRewardPersistenceEventV1
		severity = security.SeverityCritical
		message = "Miner-token reward persistence rejected."
	case "REJECTED":
		eventType = TokenMinerRewardRejectedEventV1
		severity = security.SeverityHigh
		message = "Miner-token reward rejected."
	}

	errorClass := "NONE"
	if operationErr != nil {
		switch {
		case errors.Is(operationErr, ErrMinerTokenPersistenceV1):
			errorClass = "MINER_TOKEN_PERSISTENCE"
		case errors.Is(operationErr, ErrMinerTokenFinalizedHeightConflictV1):
			errorClass = "MINER_TOKEN_FINALIZED_CONFLICT"
		case errors.Is(operationErr, ErrMinerTokenBlockIdentityV1):
			errorClass = "MINER_TOKEN_BLOCK_IDENTITY"
		default:
			errorClass = "MINER_TOKEN_REWARD_ERROR"
		}
	}

	metadata := map[string]any{
		"model":               TokenAuthorizationModelLegacyV1,
		"operation":           TokenOperationMinerRewardV1,
		"outcome":             outcome,
		"symbol":              symbol,
		"blockHeight":         height,
		"blockHash":           blockHash,
		"reward":              reward,
		"confirmationDepth":   confirmationDepth,
		"requiredFinality":    MinerTokenFinalityDepthV1,
		"errorClass":          errorClass,
		"aiReadOnly":          true,
		"advisoryOnly":        true,
		"authorizesOperation": false,
		"changesTokenState":   false,
		"changesRegistry":     false,
		"changesChain":        false,
		"changesConsensus":    false,
	}

	event := security.NewEvent(
		eventType,
		severity,
		"token",
		minerAddress,
		message,
		metadata,
	)

	_ = security.AppendEvent(
		tokenAuthorizationEventDirectoryV1(),
		event,
	)
}

// observeTokenAuthorizationV1 writes advisory security metadata only.
// It never authorizes an operation, changes token state, stores key
// material, or gives the AI permission to execute an operation.
func observeTokenAuthorizationV1(
	model string,
	operation string,
	caller string,
	domain string,
	nonce uint64,
	stage string,
	operationErr error,
) {
	eventType := TokenAuthorizationAcceptedEventV1
	severity := security.SeverityInfo
	message := "Signed token authorization accepted."

	if operationErr != nil {
		switch {
		case errors.Is(
			operationErr,
			ErrTokenAuthorizationReplay,
		):
			eventType =
				TokenAuthorizationReplayRejectedEventV1
			severity = security.SeverityHigh
			message =
				"Token authorization replay was rejected."

		case stage == tokenAuthorizationStageOperationV1:
			eventType = TokenOperationRejectedEventV1
			severity = security.SeverityWarning
			message =
				"Authorized token operation was rejected."

		default:
			eventType =
				TokenAuthorizationRejectedEventV1
			severity = security.SeverityHigh
			message =
				"Signed token authorization was rejected."
		}
	}

	domainDigest := sha256.Sum256([]byte(domain))

	metadata := map[string]any{
		"model":     model,
		"operation": operation,
		"nonce":     nonce,
		"stage":     stage,
		"errorClass": tokenAuthorizationErrorClassV1(
			operationErr,
		),
		"domainSha256": hex.EncodeToString(
			domainDigest[:],
		),
		"aiReadOnly":          true,
		"advisoryOnly":        true,
		"authorizesOperation": false,
	}

	event := security.NewEvent(
		eventType,
		severity,
		"token",
		caller,
		message,
		metadata,
	)

	// Security observation must never change the operation result.
	_ = security.AppendEvent(
		tokenAuthorizationEventDirectoryV1(),
		event,
	)
}
