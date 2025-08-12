// token/miner.go
package token

import (
	"fmt"
	"sync"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
)

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

// MinerTokenBridge: miner.OnBlock → token mint köprüsü
type MinerTokenBridge struct {
	cfg MinerTokenConfig

	reg *Registry
	tok *Token

	mu sync.Mutex
}

// NewMinerTokenBridge: registry yükler; token yoksa oluşturup kaydeder.
func NewMinerTokenBridge(cfg MinerTokenConfig) (*MinerTokenBridge, error) {
	if cfg.Symbol == "" || cfg.Owner == "" {
		return nil, fmt.Errorf("token/miner: Symbol ve Owner zorunludur")
	}
	if cfg.RewardPerBlock == 0 {
		return nil, fmt.Errorf("token/miner: RewardPerBlock > 0 olmalı")
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

	return &MinerTokenBridge{
		cfg: cfg,
		reg: reg,
		tok: tok,
	}, nil
}

// onBlock: miner.Options.OnBlock içine verilecek callback.
// Her blokta block.Miner adresine RewardPerBlock kadar token mint eder.
func (b *MinerTokenBridge) onBlock(blk *blockchain.Block, _ miner.MiningStatus) {
	if blk == nil || blk.Miner == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	// Sadece owner mint edebilir
	if err := b.tok.Mint(b.cfg.Owner, blk.Miner, b.cfg.RewardPerBlock); err != nil {
		// Sessiz hatayı loglamak istersen fmt.Printf ile yazabilirsin:
		// fmt.Printf("[token/miner] mint hata: %v\n", err)
		return
	}
	_ = b.reg.Save(b.cfg.SaveDir)
}

// Options: miner.Start(...) çağrısına verilecek opsiyonları döndürür.
// İstersen Broadcaster vb. dışarıdan set edebilirsin.
func (b *MinerTokenBridge) Options() miner.Options {
	return miner.Options{
		OnBlock:     b.onBlock,
		OnError:     nil, // istersen burada da log atabilirsin
		OnTick:      nil,
		Interval:    0,   // miner tarafındaki varsayılanı kullan
		Broadcaster: nil, // dışarıdan ekleyebilirsin
	}
}

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
	bridge, err := NewMinerTokenBridge(cfg)
	if err != nil {
		return nil, err
	}

	// Köprü opsiyonları + kullanıcı opsiyonlarını birleştir
	opts := bridge.Options()
	if len(userOpts) > 0 {
		u := userOpts[0]
		// OnBlock birleştirme: önce köprü, sonra kullanıcı callback’i
		if u.OnBlock != nil {
			userOnBlock := u.OnBlock
			opts.OnBlock = func(b *blockchain.Block, st miner.MiningStatus) {
				bridge.onBlock(b, st)
				userOnBlock(b, st)
			}
		}
		if u.OnError != nil {
			opts.OnError = u.OnError
		}
		if u.OnTick != nil {
			opts.OnTick = u.OnTick
		}
		if u.Interval > 0 {
			opts.Interval = u.Interval
		}
		if u.Broadcaster != nil {
			opts.Broadcaster = u.Broadcaster
		}
	}

	if err := miner.Start(bc, minerAddress, difficulty, opts); err != nil {
		return nil, err
	}
	return bridge, nil
}

// AttachToRunningMiner: halihazırda çalışan miner’a köprüyü eklemek istersen,
// miner’ı durdurmadan yapmak mümkün olmadığı için genellikle önerilmez.
// Bunun yerine miner.Stop → miner.Start(..., bridge.Options()) yapman sağlıklı.
// Yine de minimal bir yardımcı bırakıyorum (kullanıcı kendi kontrolünde birleştirir).
func AttachToRunningMiner(_ *MinerTokenBridge) {
	// Boş: aktif miner süreç içinde callback değiştirme desteği yok.
	// İhtiyaç olursa miner paketine "UpdateOptions" gibi bir API ekleyebiliriz.
}
