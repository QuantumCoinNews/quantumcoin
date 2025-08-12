package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config: QuantumCoin temel ayarları
type Config struct {
	// Chain & Monetary Policy
	Symbol                string `json:"symbol"`                  // "QC"
	InitialReward         int    `json:"initial_reward"`          // 50
	TotalSupply           int    `json:"total_supply"`            // 25_500_000 (0 = sınırsız)
	GenesisUnix           int64  `json:"genesis_unix"`            // 2024-09-01 00:00:00 UTC örn.
	HalvingIntervalSecs   int64  `json:"halving_interval_secs"`   // 2 yıl
	MiningPeriodSecs      int64  `json:"mining_period_secs"`      // 10 yıl (0 = sınırsız)
	TargetBlockTimeSecs   int    `json:"target_block_time_secs"`  // 30 (opsiyonel, PoW ayarı için)
	HalvingPeriodBlocks   int    `json:"halving_period_blocks"`   // Blok bazlı halving için (opsiyonel)
	DefaultDifficultyBits int    `json:"default_difficulty_bits"` // 16

	// Coinbase olgunlaşma derinliği (blok cinsinden)
	CoinbaseMaturity int `json:"coinbase_maturity"` // default 10

	// Reward split yüzdeleri
	RewardPctMiner int `json:"reward_pct_miner"` // 70
	RewardPctStake int `json:"reward_pct_stake"` // 10
	RewardPctDev   int `json:"reward_pct_dev"`   // 10
	RewardPctBurn  int `json:"reward_pct_burn"`  // 5
	// Community yüzdesi kalan üzerinden hesaplanır (>=0)

	// (YENİ) Split ödemeleri için hedef adresler (boşsa miner'a eklenir)
	RewardAddrMiner     string `json:"reward_addr_miner"`
	RewardAddrStake     string `json:"reward_addr_stake"`
	RewardAddrDev       string `json:"reward_addr_dev"`
	RewardAddrBurn      string `json:"reward_addr_burn"`
	RewardAddrCommunity string `json:"reward_addr_community"`

	// Networking
	HTTPPort  string   `json:"http_port"` // ":8081"
	P2PPort   string   `json:"p2p_port"`  // ":3001"
	BootPeers []string `json:"boot_peers"`

	// Storage
	ChainFile  string `json:"chain_file"`  // "chain_data.dat"
	BonusFile  string `json:"bonus_file"`  // "bonus_store.json"
	WalletFile string `json:"wallet_file"` // "wallet_data.json"

	// Misc
	LogLevel string `json:"log_level"` // "info","debug"
}

// ---- Defaults ----

func Default() *Config {
	// 2024-09-01 00:00:00 UTC
	const genesisUnix = 1725158400
	return &Config{
		Symbol:                "QC",
		InitialReward:         50,
		TotalSupply:           25_500_000,
		GenesisUnix:           genesisUnix,
		HalvingIntervalSecs:   int64(2 * 365 * 24 * 60 * 60),  // 2 yıl
		MiningPeriodSecs:      int64(10 * 365 * 24 * 60 * 60), // 10 yıl
		TargetBlockTimeSecs:   30,
		HalvingPeriodBlocks:   0, // kullanmıyorsan 0 bırak
		DefaultDifficultyBits: 16,

		CoinbaseMaturity: 10,

		RewardPctMiner: 70,
		RewardPctStake: 10,
		RewardPctDev:   10,
		RewardPctBurn:  5,

		RewardAddrMiner:     "",
		RewardAddrStake:     "",
		RewardAddrDev:       "",
		RewardAddrBurn:      "",
		RewardAddrCommunity: "",

		HTTPPort:  ":8081",
		P2PPort:   ":3001",
		BootPeers: []string{},

		ChainFile:  "chain_data.dat",
		BonusFile:  "bonus_store.json",
		WalletFile: "wallet_data.json",

		LogLevel: "info",
	}
}

// ---- Global erişim (thread-safe) ----

var (
	current *Config
	once    sync.Once
	mu      sync.RWMutex
)

// Load: ENV ve (varsa) dosyadan yükleyip tek bir Config oluşturur
// filePath "" ise, sadece ENV + Defaults kullanılır.
func Load(filePath string) (*Config, error) {
	var err error
	once.Do(func() {
		cfg := Default()
		applyEnv(cfg) // ENV > Defaults

		if filePath != "" {
			if _, statErr := os.Stat(filePath); statErr == nil {
				if loadErr := loadFromFile(filePath, cfg); loadErr != nil {
					err = loadErr
					return
				}
			}
		}

		if vErr := cfg.Validate(); vErr != nil {
			err = vErr
			return
		}

		mu.Lock()
		current = cfg
		mu.Unlock()
	})
	if err != nil {
		return nil, err
	}
	return Current(), nil
}

// Current: aktif konfigürasyonu döndürür (Load çağrılmış olmalı)
func Current() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if current == nil {
		// Güvenli varsayılan (testler için)
		return Default()
	}
	// shallow copy
	cpy := *current
	return &cpy
}

// Set: testlerde/config override için
func Set(c *Config) {
	if c == nil {
		return
	}
	_ = c.Validate()
	mu.Lock()
	current = c
	mu.Unlock()
}

// Validate: mantıksal doğrulama
func (c *Config) Validate() error {
	if c.InitialReward < 0 {
		return errors.New("initial_reward cannot be negative")
	}
	if c.TotalSupply < 0 {
		return errors.New("total_supply cannot be negative")
	}
	if c.HalvingIntervalSecs < 0 {
		return errors.New("halving_interval_secs cannot be negative")
	}
	if c.MiningPeriodSecs < 0 {
		return errors.New("mining_period_secs cannot be negative")
	}
	if c.DefaultDifficultyBits <= 0 || c.DefaultDifficultyBits > 255 {
		return errors.New("default_difficulty_bits must be 1..255")
	}
	if c.CoinbaseMaturity < 0 {
		return errors.New("coinbase_maturity cannot be negative")
	}
	if c.RewardPctMiner < 0 || c.RewardPctStake < 0 || c.RewardPctDev < 0 || c.RewardPctBurn < 0 {
		return errors.New("reward percentages cannot be negative")
	}
	if c.RewardPctMiner+c.RewardPctStake+c.RewardPctDev+c.RewardPctBurn > 100 {
		fmt.Println("[config] warning: reward percentages sum to >100")
	}
	if c.HTTPPort == "" || c.P2PPort == "" {
		return errors.New("ports cannot be empty")
	}
	return nil
}

// SaveToFile: config'i JSON olarak yazar (pretty)
func (c *Config) SaveToFile(filePath string) error {
	if filePath == "" {
		filePath = "config.json"
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, b, 0644)
}

// ---- Helpers ----

// HalvingIntervalBlocksComputed: TargetBlockTimeSecs ile yaklaşık blok sayısı
func (c *Config) HalvingIntervalBlocksComputed() int {
	if c.TargetBlockTimeSecs <= 0 || c.HalvingIntervalSecs <= 0 {
		return 0
	}
	return int((time.Duration(c.HalvingIntervalSecs) * time.Second) / (time.Duration(c.TargetBlockTimeSecs) * time.Second))
}

// MiningEndsAt: zaman bazlı madencilik bitiş zamanı (epoch), 0 = sınırsız
func (c *Config) MiningEndsAt() int64 {
	if c.MiningPeriodSecs <= 0 || c.GenesisUnix <= 0 {
		return 0
	}
	return c.GenesisUnix + c.MiningPeriodSecs
}

// ---- internal: ENV ve Dosya yükleyicileri ----

func loadFromFile(path string, into *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse %s failed: %w", path, err)
	}

	// Sıfır/boş olmayan değerlerle override et
	merge(into, &fileCfg)
	return nil
}

func merge(base, src *Config) {
	if src.Symbol != "" {
		base.Symbol = src.Symbol
	}
	if src.InitialReward != 0 {
		base.InitialReward = src.InitialReward
	}
	if src.TotalSupply != 0 {
		base.TotalSupply = src.TotalSupply
	}
	if src.GenesisUnix != 0 {
		base.GenesisUnix = src.GenesisUnix
	}
	if src.HalvingIntervalSecs != 0 {
		base.HalvingIntervalSecs = src.HalvingIntervalSecs
	}
	if src.MiningPeriodSecs != 0 {
		base.MiningPeriodSecs = src.MiningPeriodSecs
	}
	if src.TargetBlockTimeSecs != 0 {
		base.TargetBlockTimeSecs = src.TargetBlockTimeSecs
	}
	if src.HalvingPeriodBlocks != 0 {
		base.HalvingPeriodBlocks = src.HalvingPeriodBlocks
	}
	if src.DefaultDifficultyBits != 0 {
		base.DefaultDifficultyBits = src.DefaultDifficultyBits
	}
	if src.CoinbaseMaturity != 0 {
		base.CoinbaseMaturity = src.CoinbaseMaturity
	}
	if src.RewardPctMiner != 0 {
		base.RewardPctMiner = src.RewardPctMiner
	}
	if src.RewardPctStake != 0 {
		base.RewardPctStake = src.RewardPctStake
	}
	if src.RewardPctDev != 0 {
		base.RewardPctDev = src.RewardPctDev
	}
	if src.RewardPctBurn != 0 {
		base.RewardPctBurn = src.RewardPctBurn
	}

	// (YENİ) split adresleri
	if src.RewardAddrMiner != "" {
		base.RewardAddrMiner = src.RewardAddrMiner
	}
	if src.RewardAddrStake != "" {
		base.RewardAddrStake = src.RewardAddrStake
	}
	if src.RewardAddrDev != "" {
		base.RewardAddrDev = src.RewardAddrDev
	}
	if src.RewardAddrBurn != "" {
		base.RewardAddrBurn = src.RewardAddrBurn
	}
	if src.RewardAddrCommunity != "" {
		base.RewardAddrCommunity = src.RewardAddrCommunity
	}

	if src.HTTPPort != "" {
		base.HTTPPort = src.HTTPPort
	}
	if src.P2PPort != "" {
		base.P2PPort = src.P2PPort
	}
	if len(src.BootPeers) > 0 {
		base.BootPeers = append([]string(nil), src.BootPeers...)
	}
	if src.ChainFile != "" {
		base.ChainFile = src.ChainFile
	}
	if src.BonusFile != "" {
		base.BonusFile = src.BonusFile
	}
	if src.WalletFile != "" {
		base.WalletFile = src.WalletFile
	}
	if src.LogLevel != "" {
		base.LogLevel = src.LogLevel
	}
}

func applyEnv(c *Config) {
	// Yardımcılar
	envInt := func(key string, def int) int {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return def
	}
	envInt64 := func(key string, def int64) int64 {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				return n
			}
		}
		return def
	}
	envStr := func(key string, def string) string {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
		return def
	}
	envCSV := func(key string, def []string) []string {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			parts := strings.Split(v, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return def
	}

	// ENV key'leri (QC_* prefix)
	c.Symbol = envStr("QC_SYMBOL", c.Symbol)
	c.InitialReward = envInt("QC_INITIAL_REWARD", c.InitialReward)
	c.TotalSupply = envInt("QC_TOTAL_SUPPLY", c.TotalSupply)
	c.GenesisUnix = envInt64("QC_GENESIS_UNIX", c.GenesisUnix)
	c.HalvingIntervalSecs = envInt64("QC_HALVING_INTERVAL_SECS", c.HalvingIntervalSecs)
	c.MiningPeriodSecs = envInt64("QC_MINING_PERIOD_SECS", c.MiningPeriodSecs)
	c.TargetBlockTimeSecs = envInt("QC_TARGET_BLOCK_TIME_SECS", c.TargetBlockTimeSecs)
	c.HalvingPeriodBlocks = envInt("QC_HALVING_PERIOD_BLOCKS", c.HalvingPeriodBlocks)
	c.DefaultDifficultyBits = envInt("QC_DEFAULT_DIFFICULTY_BITS", c.DefaultDifficultyBits)

	c.CoinbaseMaturity = envInt("QC_COINBASE_MATURITY", c.CoinbaseMaturity)

	c.RewardPctMiner = envInt("QC_REWARD_PCT_MINER", c.RewardPctMiner)
	c.RewardPctStake = envInt("QC_REWARD_PCT_STAKE", c.RewardPctStake)
	c.RewardPctDev = envInt("QC_REWARD_PCT_DEV", c.RewardPctDev)
	c.RewardPctBurn = envInt("QC_REWARD_PCT_BURN", c.RewardPctBurn)

	// (YENİ) split adresleri
	c.RewardAddrMiner = envStr("QC_REWARD_ADDR_MINER", c.RewardAddrMiner)
	c.RewardAddrStake = envStr("QC_REWARD_ADDR_STAKE", c.RewardAddrStake)
	c.RewardAddrDev = envStr("QC_REWARD_ADDR_DEV", c.RewardAddrDev)
	c.RewardAddrBurn = envStr("QC_REWARD_ADDR_BURN", c.RewardAddrBurn)
	c.RewardAddrCommunity = envStr("QC_REWARD_ADDR_COMMUNITY", c.RewardAddrCommunity)

	c.HTTPPort = envStr("QC_HTTP_PORT", c.HTTPPort)
	c.P2PPort = envStr("QC_P2P_PORT", c.P2PPort)
	c.BootPeers = envCSV("QC_BOOT_PEERS", c.BootPeers)

	c.ChainFile = envStr("QC_CHAIN_FILE", c.ChainFile)
	c.BonusFile = envStr("QC_BONUS_FILE", c.BonusFile)
	c.WalletFile = envStr("QC_WALLET_FILE", c.WalletFile)

	c.LogLevel = envStr("QC_LOG_LEVEL", c.LogLevel)
}
