package config

import qconsensus "quantumcoin/consensus"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//
// ===== Backward-compat shims (eski kod uyumluluğu) =====
//

// QC documents the QuantumCoin component used by this module.
// Deprecated: use consensus.CanonicalAtomsPerQCV1. This alias has the same canonical scale.
const QC int64 = qconsensus.CanonicalAtomsPerQCV1

// AnnualBonusPerYearQC is a retired compatibility value.
// QMA-0047: the annual CMD mining bonus is cancelled and cannot be enabled by ENV.
const AnnualBonusPerYearQC int64 = 0

// GenesisTime documents the QuantumCoin component used by this module.
// Eski kod bazı yerlerde time.Time olarak GenesisTime bekliyor.
var GenesisTime time.Time

// BlocksPerYear documents the QuantumCoin component used by this module.
// BlocksPerYear: hedef blok süresine göre yaklaşık blok/yıl (int64 döner)
func BlocksPerYear() int64 {
	c := Current()
	secs := c.TargetBlockTimeSecs
	if secs <= 0 {
		secs = qconsensus.CanonicalBlockTimeSecondsV1
	}
	return int64((365 * 24 * 3600) / secs)
}

//
// ===== Yeni yapı =====
//

// Config documents the QuantumCoin component used by this module.
// Config: QuantumCoin temel ayarları
type Config struct {
	// Miner is local operator configuration, not a consensus reward allocation.
	Miner              MinerConfig `json:"miner"`
	MinerAddressSource string      `json:"-"`
	// --- Chain & Monetary Policy ---
	Symbol                string `json:"symbol"`
	InitialReward         int    `json:"initial_reward"`
	TotalSupply           int    `json:"total_supply"`
	GenesisUnix           int64  `json:"genesis_unix"`
	GenesisProfile        string `json:"genesis_profile,omitempty"`
	HalvingIntervalSecs   int64  `json:"halving_interval_secs"`
	MiningPeriodSecs      int64  `json:"mining_period_secs"`
	TargetBlockTimeSecs   int    `json:"target_block_time_secs"`
	HalvingPeriodBlocks   int    `json:"halving_period_blocks"`
	DefaultDifficultyBits int    `json:"default_difficulty_bits"`

	// --- Coinbase maturity (in blocks) ---
	CoinbaseMaturity int `json:"coinbase_maturity"`

	// --- Legacy reward proposal fields: compatibility input only; consensus ignores them. ---
	RewardPctMiner int `json:"reward_pct_miner"`
	RewardPctStake int `json:"reward_pct_stake"`
	RewardPctDev   int `json:"reward_pct_dev"`
	RewardPctBurn  int `json:"reward_pct_burn"`
	// The implemented community coinbase share is zero; see MINING_REWARD_CONTRACT.md.

	// --- Kanonik adres alanları ---
	DevFundAddress       string `json:"dev_fund_address"`
	StakePoolAddress     string `json:"stake_pool_address"`
	CommunityPoolAddress string `json:"community_pool_address"`
	BurnAddress          string `json:"burn_address"`

	// --- Economy pool addresses (ECONOMY_POLICY_V1) ---
	MiningPoolAddress string `json:"mining_pool_address"`
	// --- Eski/alternatif adres alanları (normalize edilecek) ---
	RewardAddrMiner     string `json:"reward_addr_miner"`
	RewardAddrStake     string `json:"reward_addr_stake"`
	RewardAddrDev       string `json:"reward_addr_dev"`
	RewardAddrBurn      string `json:"reward_addr_burn"`
	RewardAddrCommunity string `json:"reward_addr_community"`
	// --- Premine / Founder allocation ---
	// Founder allocation target: 5% total.
	// Planned split: 1% liquid for listing/operations, 4% locked/vesting.
	PreminePercent               int    `json:"premine_percent"`        // default: 5
	PremineLiquidPercent         int    `json:"premine_liquid_percent"` // default: 1
	PremineLockedPercent         int    `json:"premine_locked_percent"` // default: 4
	PremineAddress               string `json:"premine_address"`        // fallback founder address
	PremineLiquidAddress         string `json:"premine_liquid_address"`
	PremineLockedAddress         string `json:"premine_locked_address"`
	FounderVestingCliffMonths    int    `json:"founder_vesting_cliff_months"`    // default: 12
	FounderVestingDurationMonths int    `json:"founder_vesting_duration_months"` // default: 48

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

type MinerConfig struct {
	Address string `json:"address"`
}

// ---- Defaults ----

func Default() *Config {
	const genesisUnix = qconsensus.CanonicalGenesisUnixV1 // 2024-09-01 00:00:00 UTC
	return &Config{
		Symbol:                "QC",
		InitialReward:         int(qconsensus.CanonicalInitialMiningRewardQCV1),
		TotalSupply:           int(qconsensus.CanonicalTotalSupplyQCV1),
		GenesisUnix:           genesisUnix,
		HalvingIntervalSecs:   qconsensus.CanonicalHalvingIntervalSecondsV1,
		MiningPeriodSecs:      qconsensus.CanonicalMiningPeriodSecondsV1,
		TargetBlockTimeSecs:   qconsensus.CanonicalBlockTimeSecondsV1,
		HalvingPeriodBlocks:   0,
		DefaultDifficultyBits: qconsensus.CanonicalDefaultDifficultyBitsV1,
		CoinbaseMaturity:      qconsensus.CanonicalCoinbaseMaturityBlocksV1,
		RewardPctMiner:        ShareMiningPct,
		RewardPctStake:        ShareStakingPct,
		RewardPctDev:          ShareDevPct,
		RewardPctBurn:         ShareBurnPct,

		DevFundAddress:       "",
		StakePoolAddress:     "",
		CommunityPoolAddress: "",
		BurnAddress:          qconsensus.CanonicalBurnSinkAddressV1,
		MiningPoolAddress:    qconsensus.CanonicalMiningEmissionPoolAddressV1,
		RewardAddrMiner:      "",
		RewardAddrStake:      "",
		RewardAddrDev:        "",
		RewardAddrBurn:       "",
		RewardAddrCommunity:  "",

		// Premine / founder defaults
		PreminePercent:               qconsensus.CanonicalFounderTotalPercentV1,
		PremineLiquidPercent:         qconsensus.CanonicalFounderLiquidPercentV1,
		PremineLockedPercent:         qconsensus.CanonicalFounderLockedPercentV1,
		PremineAddress:               "",
		PremineLiquidAddress:         "",
		PremineLockedAddress:         "",
		FounderVestingCliffMonths:    qconsensus.CanonicalFounderCliffMonthsV1,
		FounderVestingDurationMonths: qconsensus.CanonicalFounderVestingMonthsV1,
		HTTPPort:                     ":8082",
		P2PPort:                      ":3001",
		BootPeers:                    []string{},

		ChainFile:  "chain_data.dat",
		BonusFile:  "bonus_store.json",
		WalletFile: "wallet_data.json",

		LogLevel: "info",
	}
}

// ---- Global erişim (thread-safe) ----

var (
	current *Config
	mu      sync.RWMutex
)

// Load documents the QuantumCoin component used by this module.
// Load: ENV ve (varsa) dosyadan yükle; normalize et; doğrula; global ata.
func Load(filePath string) (*Config, error) {
	return loadWithDataDir(filePath, "")
}

// LoadWithDataDir anchors storage paths before validation and publication.
// Relative file/environment overrides use dataDir; absolute paths stay intact.
// It never changes the process working directory or creates directories.
func LoadWithDataDir(filePath, dataDir string) (*Config, error) {
	base, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	return loadWithDataDir(filePath, base)
}

func loadWithDataDir(filePath, dataDir string) (*Config, error) {
	if err := ValidateProcessRuntimeEnvironmentV1("node"); err != nil {
		return nil, err
	}
	cfg := Default()

	// Configuration precedence:
	// defaults -> file -> environment -> normalize -> validate.
	if filePath != "" {
		if _, statErr := os.Stat(filePath); statErr == nil {
			if err := loadFromFile(filePath, cfg); err != nil {
				return nil, err
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return nil, fmt.Errorf(
				"stat config %s failed: %w",
				filePath,
				statErr,
			)
		}
	}

	// Deployment/runtime environment has final precedence.
	applyEnv(cfg)
	if dataDir != "" {
		for _, path := range []*string{&cfg.ChainFile, &cfg.BonusFile, &cfg.WalletFile} {
			if *path != "" && !filepath.IsAbs(*path) {
				*path = filepath.Join(dataDir, *path)
			}
		}
	}
	cfg.normalizeRewardAddresses()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := ValidateRuntimeHTTPBindV1(cfg); err != nil {
		return nil, err
	}

	mu.Lock()
	current = cfg

	if cfg.GenesisUnix > 0 {
		GenesisTime = time.Unix(
			cfg.GenesisUnix,
			0,
		).UTC()
	} else {
		GenesisTime = time.Unix(
			Default().GenesisUnix,
			0,
		).UTC()
	}

	result := *cfg
	result.BootPeers = append(
		[]string(nil),
		cfg.BootPeers...,
	)

	mu.Unlock()

	return &result, nil
}

// Current documents the QuantumCoin component used by this module.
// Current: aktif konfigürasyon
func Current() *Config {
	mu.RLock()
	defer mu.RUnlock()

	if current == nil {
		return Default()
	}

	cpy := *current
	cpy.BootPeers = append(
		[]string(nil),
		current.BootPeers...,
	)

	return &cpy
}

// SetValidated validates and defensively copies the supplied
// configuration before replacing the process-global configuration.
func SetValidated(c *Config) error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	candidate := *c
	candidate.BootPeers = append(
		[]string(nil),
		c.BootPeers...,
	)

	if err := candidate.Validate(); err != nil {
		return err
	}

	mu.Lock()
	current = &candidate

	if candidate.GenesisUnix > 0 {
		GenesisTime = time.Unix(
			candidate.GenesisUnix,
			0,
		).UTC()
	}

	mu.Unlock()

	return nil
}

// Set preserves the legacy API while refusing invalid or
// mainnet-immutable configuration overrides.
func Set(c *Config) {
	_ = SetValidated(c)
}

// ====== EK: Community yüzdesi ve “ana cüzdan” çözümleyici ======

// CommunityPercent documents the QuantumCoin component used by this module.
// CommunityPercent returns the implemented block reward community share (zero).
// Legacy reward_pct fields are compatibility input only and do not direct coinbase outputs.
func (c *Config) CommunityPercent() int {
	return ShareCommunityPct
}

// ResolveCommunityAddress documents the QuantumCoin component used by this module.
// ResolveCommunityAddress:
// community_pool_address boşsa öncelik sırası:
// 1) PremineAddress (ANA CÜZDAN)
// 2) DevFundAddress
// 3) (boş)
func (c *Config) ResolveCommunityAddress() string {
	if strings.TrimSpace(c.CommunityPoolAddress) != "" {
		return c.CommunityPoolAddress
	}
	if strings.TrimSpace(c.PremineAddress) != "" {
		return c.PremineAddress
	}
	if strings.TrimSpace(c.DevFundAddress) != "" {
		return c.DevFundAddress
	}
	return ""
}

// Validate documents the QuantumCoin component used by this module.
// Validate: mantıksal doğrulama
func (c *Config) Validate() error {
	if err := ValidateChainFileSeparationV1(c); err != nil {
		return err
	}
	if err := ValidateMainnetImmutableConfigV1(c); err != nil {
		return err
	}
	if err := ValidateFounderAddressBindingV1(c); err != nil {
		return err
	}
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
	if err := ValidateRewardRolesV1(c); err != nil {
		return err
	}

	if c.PreminePercent < 0 || c.PreminePercent > 100 {
		return errors.New("premine_percent must be 0..100")
	}
	if c.PremineLiquidPercent < 0 || c.PremineLiquidPercent > 100 {
		return errors.New("premine_liquid_percent must be 0..100")
	}
	if c.PremineLockedPercent < 0 || c.PremineLockedPercent > 100 {
		return errors.New("premine_locked_percent must be 0..100")
	}
	if c.PremineLiquidPercent+c.PremineLockedPercent != c.PreminePercent {
		return errors.New("premine liquid+locked percent must equal premine_percent")
	}
	if c.FounderVestingCliffMonths < 0 {
		return errors.New("founder_vesting_cliff_months cannot be negative")
	}
	if c.FounderVestingDurationMonths < 0 {
		return errors.New("founder_vesting_duration_months cannot be negative")
	}
	if c.HTTPPort == "" || c.P2PPort == "" {
		return errors.New("ports cannot be empty")
	}
	return nil
}

// SaveToFile documents the QuantumCoin component used by this module.
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
	if into == nil {
		return errors.New("config destination cannot be nil")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := rejectDuplicateConfigKeysV1(file); err != nil {
		return fmt.Errorf("parse %s failed: %w", path, err)
	}

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	// Decode into the structure already populated with defaults.
	// Omitted fields retain their defaults. Explicit zero values
	// overwrite defaults correctly.
	if err := decoder.Decode(into); err != nil {
		return fmt.Errorf(
			"parse %s failed: %w",
			path,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf(
				"parse %s failed: multiple JSON values",
				path,
			)
		}

		return fmt.Errorf(
			"parse %s trailing data failed: %w",
			path,
			err,
		)
	}

	return nil
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
	c.Miner.Address = strings.TrimSpace(c.Miner.Address)
	if c.Miner.Address != "" {
		c.MinerAddressSource = "config.miner.address"
	}
	if miner := strings.TrimSpace(os.Getenv("QC_MINER")); miner != "" {
		c.Miner.Address = miner
		c.MinerAddressSource = "QC_MINER"
	}
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
	c.MiningPoolAddress = envStr("QC_MINING_POOL_ADDRESS", c.MiningPoolAddress)
	// Founder / Premine
	c.PreminePercent = envInt("QC_PREMINE_PERCENT", c.PreminePercent)
	c.PremineLiquidPercent = envInt("QC_PREMINE_LIQUID_PERCENT", c.PremineLiquidPercent)
	c.PremineLockedPercent = envInt("QC_PREMINE_LOCKED_PERCENT", c.PremineLockedPercent)
	c.PremineAddress = envStr("QC_PREMINE_ADDRESS", c.PremineAddress)
	c.PremineLiquidAddress = envStr("QC_PREMINE_LIQUID_ADDRESS", c.PremineLiquidAddress)
	c.PremineLockedAddress = envStr("QC_PREMINE_LOCKED_ADDRESS", c.PremineLockedAddress)
	c.FounderVestingCliffMonths = envInt("QC_FOUNDER_VESTING_CLIFF_MONTHS", c.FounderVestingCliffMonths)
	c.FounderVestingDurationMonths = envInt("QC_FOUNDER_VESTING_DURATION_MONTHS", c.FounderVestingDurationMonths)

	c.HTTPPort = envStr("QC_HTTP_PORT", c.HTTPPort)
	c.P2PPort = envStr("QC_P2P_PORT", c.P2PPort)
	c.BootPeers = envCSV("QC_BOOT_PEERS", c.BootPeers)

	c.ChainFile = envStr("QC_CHAIN_FILE", c.ChainFile)
	c.BonusFile = envStr("QC_BONUS_FILE", c.BonusFile)
	c.WalletFile = envStr("QC_WALLET_FILE", c.WalletFile)

	c.LogLevel = envStr("QC_LOG_LEVEL", c.LogLevel)
	// QMA-0047: QC_ANNUAL_BONUS_QC is retired and intentionally ignored.
}

// Eski adres alanlarından kanoniğe taşıma + premine/community fallback
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
	if c.PremineLiquidAddress == "" && c.PremineAddress != "" {
		c.PremineLiquidAddress = c.PremineAddress
	}
	if c.PremineLockedAddress == "" && c.PremineAddress != "" {
		c.PremineLockedAddress = c.PremineAddress
	}
	// ADDED: Community adresi yoksa ana cüzdana (premine) ya da dev fona bağla
	if strings.TrimSpace(c.CommunityPoolAddress) == "" {
		if strings.TrimSpace(c.PremineAddress) != "" {
			c.CommunityPoolAddress = c.PremineAddress
		} else if strings.TrimSpace(c.DevFundAddress) != "" {
			c.CommunityPoolAddress = c.DevFundAddress
		}
	}
}
