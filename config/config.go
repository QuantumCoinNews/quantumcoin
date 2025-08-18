// config/config.go
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

//
// ===== Backward-compat shims (eski kod uyumluluğu) =====
//

// 1 QC = 100,000,000 "atoms"
const QC int64 = 100_000_000

// Eski kod bu sabiti bekliyor; ENV ile override edilebilir (QC_ANNUAL_BONUS_QC).
var AnnualBonusPerYearQC int64 = 100

// Eski kod bazı yerlerde time.Time olarak GenesisTime bekliyor.
var GenesisTime time.Time

// BlocksPerYear: hedef blok süresine göre yaklaşık blok/yıl (int64 döner)
func BlocksPerYear() int64 {
	c := Current()
	secs := c.TargetBlockTimeSecs
	if secs <= 0 {
		secs = 30
	}
	return int64((365 * 24 * 3600) / secs)
}

//
// ===== Yeni yapı =====
//

// Config: QuantumCoin temel ayarları
type Config struct {
	// --- Chain & Monetary Policy ---
	Symbol                string `json:"symbol"`
	InitialReward         int    `json:"initial_reward"`
	TotalSupply           int    `json:"total_supply"`
	GenesisUnix           int64  `json:"genesis_unix"`
	HalvingIntervalSecs   int64  `json:"halving_interval_secs"`
	MiningPeriodSecs      int64  `json:"mining_period_secs"`
	TargetBlockTimeSecs   int    `json:"target_block_time_secs"`
	HalvingPeriodBlocks   int    `json:"halving_period_blocks"`
	DefaultDifficultyBits int    `json:"default_difficulty_bits"`

	// --- Coinbase maturity (in blocks) ---
	CoinbaseMaturity int `json:"coinbase_maturity"`

	// --- Reward split percentages ---
	RewardPctMiner int `json:"reward_pct_miner"`
	RewardPctStake int `json:"reward_pct_stake"`
	RewardPctDev   int `json:"reward_pct_dev"`
	RewardPctBurn  int `json:"reward_pct_burn"`
	// Community = 100 - (yukarıdakilerin toplamı)

	// --- Kanonik adres alanları ---
	DevFundAddress       string `json:"dev_fund_address"`
	StakePoolAddress     string `json:"stake_pool_address"`
	CommunityPoolAddress string `json:"community_pool_address"`
	BurnAddress          string `json:"burn_address"`

	// --- Eski/alternatif adres alanları (normalize edilecek) ---
	RewardAddrMiner     string `json:"reward_addr_miner"`
	RewardAddrStake     string `json:"reward_addr_stake"`
	RewardAddrDev       string `json:"reward_addr_dev"`
	RewardAddrBurn      string `json:"reward_addr_burn"`
	RewardAddrCommunity string `json:"reward_addr_community"`

	// --- Premine (ANA CÜZDAN) ---
	PreminePercent int    `json:"premine_percent"` // varsayılan: 12
	PremineAddress string `json:"premine_address"` // boşsa DevFundAddress kullanılır

	// --- Networking ---
	HTTPPort  string   `json:"http_port"`
	P2PPort   string   `json:"p2p_port"`
	BootPeers []string `json:"boot_peers"`

	// --- Storage ---
	ChainFile  string `json:"chain_file"`
	BonusFile  string `json:"bonus_file"`
	WalletFile string `json:"wallet_file"`

	// --- Misc ---
	LogLevel string `json:"log_level"`
}

// ---- Defaults ----

func Default() *Config {
	const genesisUnix = 1725158400 // 2024-09-01 00:00:00 UTC
	return &Config{
		Symbol:                "QC",
		InitialReward:         50,
		TotalSupply:           25_500_000,
		GenesisUnix:           genesisUnix,
		HalvingIntervalSecs:   int64(2 * 365 * 24 * 60 * 60),
		MiningPeriodSecs:      int64(10 * 365 * 24 * 60 * 60),
		TargetBlockTimeSecs:   30,
		HalvingPeriodBlocks:   0,
		DefaultDifficultyBits: 16,

		CoinbaseMaturity: 10,

		RewardPctMiner: 70,
		RewardPctStake: 10,
		RewardPctDev:   10,
		RewardPctBurn:  5,

		DevFundAddress:       "",
		StakePoolAddress:     "",
		CommunityPoolAddress: "",
		BurnAddress:          "QC_BURN_SINK",

		RewardAddrMiner:     "",
		RewardAddrStake:     "",
		RewardAddrDev:       "",
		RewardAddrBurn:      "",
		RewardAddrCommunity: "",

		// Premine defaults
		PreminePercent: 12,
		PremineAddress: "",

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

// Load: ENV ve (varsa) dosyadan yükle; normalize et; doğrula; global ata.
func Load(filePath string) (*Config, error) {
	var err error
	once.Do(func() {
		cfg := Default()
		applyEnv(cfg)

		if filePath != "" {
			if _, statErr := os.Stat(filePath); statErr == nil {
				if loadErr := loadFromFile(filePath, cfg); loadErr != nil {
					err = loadErr
					return
				}
			}
		}

		cfg.normalizeRewardAddresses()

		if vErr := cfg.Validate(); vErr != nil {
			err = vErr
			return
		}

		mu.Lock()
		current = cfg
		// Back-compat: GenesisTime
		if cfg.GenesisUnix > 0 {
			GenesisTime = time.Unix(cfg.GenesisUnix, 0).UTC()
		} else {
			GenesisTime = time.Unix(Default().GenesisUnix, 0).UTC()
		}
		mu.Unlock()
	})
	if err != nil {
		return nil, err
	}
	return Current(), nil
}

// Current: aktif konfigürasyon
func Current() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if current == nil {
		return Default()
	}
	cpy := *current
	return &cpy
}

// Set: test/override
func Set(c *Config) {
	if c == nil {
		return
	}
	_ = c.Validate()
	mu.Lock()
	current = c
	if c.GenesisUnix > 0 {
		GenesisTime = time.Unix(c.GenesisUnix, 0).UTC()
	}
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
	sum := c.RewardPctMiner + c.RewardPctStake + c.RewardPctDev + c.RewardPctBurn
	if sum > 100 {
		fmt.Println("[config] warning: reward percentages sum to >100")
	}
	if sum < 100 {
		fmt.Printf("[config] info: reward percentages sum to %d%%; remaining %d%% goes to community.\n", sum, 100-sum)
	}
	if c.PreminePercent < 0 || c.PreminePercent > 100 {
		return errors.New("premine_percent must be 0..100")
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

func (c *Config) HalvingIntervalBlocksComputed() int {
	if c.TargetBlockTimeSecs <= 0 || c.HalvingIntervalSecs <= 0 {
		return 0
	}
	return int((time.Duration(c.HalvingIntervalSecs) * time.Second) /
		(time.Duration(c.TargetBlockTimeSecs) * time.Second))
}

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
	merge(into, &fileCfg)
	return nil
}

func merge(base, src *Config) {
	// Basit merge (zero-value olmayanlar)
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

	// Kanonik adresler
	if src.DevFundAddress != "" {
		base.DevFundAddress = src.DevFundAddress
	}
	if src.StakePoolAddress != "" {
		base.StakePoolAddress = src.StakePoolAddress
	}
	if src.CommunityPoolAddress != "" {
		base.CommunityPoolAddress = src.CommunityPoolAddress
	}
	if src.BurnAddress != "" {
		base.BurnAddress = src.BurnAddress
	}

	// Eski alanlar
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

	// Premine
	if src.PreminePercent != 0 {
		base.PreminePercent = src.PreminePercent
	}
	if src.PremineAddress != "" {
		base.PremineAddress = src.PremineAddress
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
	// Helpers
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

	// ENV (QC_* prefix)
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

	// Eski/alternatif adres alanları
	c.RewardAddrMiner = envStr("QC_REWARD_ADDR_MINER", c.RewardAddrMiner)
	c.RewardAddrStake = envStr("QC_REWARD_ADDR_STAKE", c.RewardAddrStake)
	c.RewardAddrDev = envStr("QC_REWARD_ADDR_DEV", c.RewardAddrDev)
	c.RewardAddrBurn = envStr("QC_REWARD_ADDR_BURN", c.RewardAddrBurn)
	c.RewardAddrCommunity = envStr("QC_REWARD_ADDR_COMMUNITY", c.RewardAddrCommunity)

	// Kanonik adresler
	c.DevFundAddress = envStr("QC_DEV_FUND_ADDRESS", c.DevFundAddress)
	c.StakePoolAddress = envStr("QC_STAKE_POOL_ADDRESS", c.StakePoolAddress)
	c.CommunityPoolAddress = envStr("QC_COMMUNITY_POOL_ADDRESS", c.CommunityPoolAddress)
	c.BurnAddress = envStr("QC_BURN_ADDRESS", c.BurnAddress)

	// Premine
	c.PreminePercent = envInt("QC_PREMINE_PERCENT", c.PreminePercent)
	c.PremineAddress = envStr("QC_PREMINE_ADDRESS", c.PremineAddress)

	c.HTTPPort = envStr("QC_HTTP_PORT", c.HTTPPort)
	c.P2PPort = envStr("QC_P2P_PORT", c.P2PPort)
	c.BootPeers = envCSV("QC_BOOT_PEERS", c.BootPeers)

	c.ChainFile = envStr("QC_CHAIN_FILE", c.ChainFile)
	c.BonusFile = envStr("QC_BONUS_FILE", c.BonusFile)
	c.WalletFile = envStr("QC_WALLET_FILE", c.WalletFile)

	c.LogLevel = envStr("QC_LOG_LEVEL", c.LogLevel)

	// Back-compat: yıllık bonus ENV override
	if v := strings.TrimSpace(os.Getenv("QC_ANNUAL_BONUS_QC")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			AnnualBonusPerYearQC = n
		}
	}
}

// Eski adres alanlarından kanoniğe taşıma + premine fallback
func (c *Config) normalizeRewardAddresses() {
	if c.DevFundAddress == "" && c.RewardAddrDev != "" {
		c.DevFundAddress = c.RewardAddrDev
	}
	if c.StakePoolAddress == "" && c.RewardAddrStake != "" {
		c.StakePoolAddress = c.RewardAddrStake
	}
	if c.CommunityPoolAddress == "" && c.RewardAddrCommunity != "" {
		c.CommunityPoolAddress = c.RewardAddrCommunity
	}
	if c.BurnAddress == "" && c.RewardAddrBurn != "" {
		c.BurnAddress = c.RewardAddrBurn
	}
	// Premine adresi boşsa DevFundAddress'tan devral
	if c.PremineAddress == "" && c.DevFundAddress != "" {
		c.PremineAddress = c.DevFundAddress
	}
}
