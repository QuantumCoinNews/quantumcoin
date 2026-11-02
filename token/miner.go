// token/miner.go

// Package token contains QuantumCoin project code for this module.
package token

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
)

// MinerTokenConfig documents the QuantumCoin component used by this module.
// MinerTokenConfig: madencilik → token mint köprüsü ayarları
type MinerTokenConfig struct {
	Symbol         string // Örn: "QC20"
	Name           string // Örn: "QuantumCoin Utility"
	Decimals       uint8  // Örn: 8
	MaxSupply      uint64 // 0 => sınırsız
	Owner          string // Token owner (mint yetkisi)
	RewardPerBlock uint64 // Her blokta mint edilecek token miktarı
	SaveDir        string // registry dosyasını yazacağın dizin ("" => çalışma dizini)
}

// MinerTokenBridge documents the QuantumCoin component used by this module.
// MinerTokenBridge: miner.OnBlock → token mint köprüsü
type MinerTokenBridge struct {
	cfg MinerTokenConfig

	chain *blockchain.Blockchain
	reg   *Registry
	tok   *Token

	mu sync.Mutex
}

// NewMinerTokenBridge documents the QuantumCoin component used by this module.
// NewMinerTokenBridge: registry yükler; token yoksa oluşturup kaydeder.
func NewMinerTokenBridge(
	cfg MinerTokenConfig,
	chains ...*blockchain.Blockchain,
) (*MinerTokenBridge, error) {
	// TOKEN_SAFE_BOUNDARY_V1_MINER_BRIDGE_GUARD
	// Token mining bridge is disabled by default.
	// It can only run when explicitly enabled by the operator:
	// QC_ENABLE_TOKEN_MINER_BRIDGE=1
	if !TokenMinerBridgeEnabled() {
		return nil, ErrTokenMinerBridgeDisabled
	}
	if cfg.Symbol == "" || cfg.Owner == "" {
		return nil, fmt.Errorf("token/miner: Symbol ve Owner zorunludur")
	}
	if cfg.RewardPerBlock == 0 {
		return nil, fmt.Errorf("token/miner: RewardPerBlock > 0 olmalı")
	}
	if len(chains) != 1 || chains[0] == nil {
		return nil, ErrMinerTokenCanonicalChainRequiredV1
	}

	reg := NewRegistry()
	if err := reg.Load(cfg.SaveDir); err != nil {
		return nil, fmt.Errorf("registry yüklenemedi: %w", err)
	}

	tok, ok := reg.Get(cfg.Symbol)
	if !ok {
		// Token kayıtlı değilse oluştur ve kaydet
		tok = New(cfg.Symbol, cfg.Name, cfg.Decimals, cfg.MaxSupply, cfg.Owner)
		if err := reg.Register(tok); err != nil {
			return nil, fmt.Errorf("token kaydı başarısız: %w", err)
		}
		if err := reg.Save(cfg.SaveDir); err != nil {
			return nil, fmt.Errorf("registry kaydedilemedi: %w", err)
		}
	}

	bridge := &MinerTokenBridge{
		cfg:   cfg,
		chain: chains[0],
		reg:   reg,
		tok:   tok,
	}

	bridge.mu.Lock()
	err := bridge.reconcilePendingMinerTokenBlocksV1()
	bridge.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return bridge, nil
}

func reportMinerTokenBridgeErrorV1(
	err error,
	onError func(error),
) {
	if err == nil {
		return
	}

	wrapped := fmt.Errorf(
		"token/miner: bridge callback failed: %w",
		err,
	)
	if onError != nil {
		onError(wrapped)
		return
	}

	log.Printf("%v", wrapped)
}

// Options documents the QuantumCoin component used by this module.
// Options: köprü ve isteğe bağlı kullanıcı callback'lerini tek hata yolunda
// birleştirerek miner.Start(...) seçeneklerini döndürür.
func (b *MinerTokenBridge) Options(
	userOpts ...miner.Options,
) miner.Options {
	var user miner.Options

	// miner options contract:
	// Merge every variadic user option instead of silently using only
	// userOpts[0]. Non-zero callback/configuration fields use last-wins
	// precedence. Animate is a plain bool, so the last supplied option
	// owns its value.
	for _, option := range userOpts {
		if option.OnBlock != nil {
			user.OnBlock = option.OnBlock
		}
		if option.OnError != nil {
			user.OnError = option.OnError
		}
		if option.OnTick != nil {
			user.OnTick = option.OnTick
		}
		if option.Interval > 0 {
			user.Interval = option.Interval
		}
		if option.Broadcaster != nil {
			user.Broadcaster = option.Broadcaster
		}

		user.Animate = option.Animate
	}

	userOnBlock := user.OnBlock
	userOnError := user.OnError

	opts := miner.Options{
		OnError:     userOnError,
		OnTick:      user.OnTick,
		Interval:    user.Interval,
		Broadcaster: user.Broadcaster,
		Animate:     user.Animate,
	}

	opts.OnBlock = func(
		block *blockchain.Block,
		status miner.MiningStatus,
	) {
		err := b.processMinerTokenBlockV1(
			block,
			status,
		)

		reportMinerTokenBridgeErrorV1(
			err,
			userOnError,
		)

		// Token reward failure does not invalidate the canonical QC block.
		if userOnBlock != nil {
			userOnBlock(block, status)
		}
	}

	return opts
}

// StartMiningWithToken documents the QuantumCoin component used by this module.
// StartMiningWithToken: tek satırda köprüyü kurar ve madenciliği başlatır.
// Not: main.go'da bc yüklendikten sonra miner.SetGlobalBlockchain(bc) çağrın mevcutsa,
// buradan doğrudan miner.Start kullanmak için gerekmez ama biz explicit bc veriyoruz.
func StartMiningWithToken(
	bc *blockchain.Blockchain,
	minerAddress string,
	difficulty int,
	cfg MinerTokenConfig,
	userOpts ...miner.Options, // (opsiyonel) kendi OnBlock/Broadcaster eklemek istersen
) (*MinerTokenBridge, error) {
	bridge, err := NewMinerTokenBridge(
		cfg,
		bc,
	)
	if err != nil {
		return nil, err
	}

	// Köprü ve kullanıcı callback'leri aynı hata kanalında birleştirilir.
	opts := bridge.Options(userOpts...)

	if err := miner.Start(bc, minerAddress, difficulty, opts); err != nil {
		return nil, err
	}
	return bridge, nil
}

var ErrAttachToRunningMinerUnsupportedV1 = errors.New(
	"token/miner: attaching bridge to a running miner is unsupported",
)

// AttachToRunningMiner fails closed because miner options/callbacks
// cannot be replaced safely while the mining loop is running.
// Stop the miner and restart it with bridge.Options(...) instead.
func AttachToRunningMiner(
	_ *MinerTokenBridge,
) error {
	return ErrAttachToRunningMinerUnsupportedV1
}
