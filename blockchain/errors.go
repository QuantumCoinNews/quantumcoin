package blockchain

import "errors"

// Genel
var (
	ErrInvalidPoW             = errors.New("invalid proof-of-work")
	ErrPrevHashMismatch       = errors.New("prev hash mismatch")
	ErrIncomingChainNotLonger = errors.New("incoming chain is not longer")
	ErrIncomingChainInvalid   = errors.New("incoming chain is invalid")
	ErrNilTransaction         = errors.New("nil transaction")
	ErrChainNotInitialized    = errors.New("blockchain not initialized (no genesis)")
)

// Coinbase / madenci
var (
	ErrMinerAddressEmpty = errors.New("miner address empty")
)

// NFT / yardımcılar
var (
	ErrNilBlockchain = errors.New("blockchain is nil")
	ErrNoBlocks      = errors.New("no blocks in chain")
)

// Transaction / fonlar
var (
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrInsufficientBalance  = errors.New("yetersiz bakiye") // mevcut metni korunur
	ErrAmountMustBePositive = errors.New("amount must be positive")
	ErrInvalidSpendableTxID = errors.New("invalid txid hex in spendable set")
)
