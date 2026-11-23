package miner

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consolefx"
)

// MiningStatus documents the QuantumCoin component used by this module.
// MiningStatus: dışa raporlanan durum
type MiningStatus struct {
	BlockHeight int
	BlockHash   []byte
	Reward      int
	Timestamp   time.Time
}

// Options documents the QuantumCoin component used by this module.
// Options: başlatma seçenekleri ve geri çağrılar
type Options struct {
	OnBlock     func(b *blockchain.Block, status MiningStatus) // Blok bulunduğunda
	OnError     func(err error)                                // Hata olduğunda
	OnTick      func()                                         // Her tur sonunda
	Interval    time.Duration                                  // Blok bulduktan sonra bekleme
	Broadcaster func(b *blockchain.Block)                      // P2P yayıncı (opsiyonel)
	Animate     bool                                           // Terminal animasyon
}

// ---- iç durum ----
var (
	state            minerState
	minerLifecycleMu sync.Mutex
	globalBC         atomic.Pointer[blockchain.Blockchain] // geriye dönük uyumluluk
)

type minerState struct {
	active   atomic.Bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopping bool
	bc       *blockchain.Blockchain
	address  string

	// requestedDifficulty <= 0 means dynamically resolve from the
	// current blockchain before every mining attempt.
	requestedDifficulty int
	difficulty          int
	opts                Options

	// görsel/istatistik
	effect   *Effect
	step     int
	hashrate float64
}

/*
-----------------------------------------------------------

	MINED BALANCE PERSIST (yalnız kazımdan gelen bakiyeyi tut)
	-----------------------------------------------------------
*/
func exeDir() string {
	self, _ := os.Executable()
	return filepath.Dir(self)
}
func minedBalancePath() string { return filepath.Join(exeDir(), "mined_balance.json") }

type minedBal struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
	Updated int64   `json:"updated"`
}

// Her blokta kazanılan ödülü kümülatif olarak dosyaya ekler.
// Tek adresi takip eder; adres değişirse sayaç sıfırlanır.
func addMinedBalance(addr string, deltaQC int) {
	addr = strings.TrimSpace(addr)
	if addr == "" || deltaQC == 0 {
		return
	}

	p := minedBalancePath()
	var mb minedBal
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &mb) // bozulmuşsa sıfırdan yazarız
	}

	if !strings.EqualFold(strings.TrimSpace(mb.Address), addr) {
		mb.Address = addr
		mb.Balance = 0
	}
	mb.Balance += float64(deltaQC)
	mb.Updated = time.Now().Unix()

	tmp := p + ".tmp"
	b, err := json.MarshalIndent(mb, "", "  ")
	if err != nil {
		log.Printf("miner: mined balance encode failed: %v", err)
		return
	}
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		log.Printf("miner: mined balance temp write failed: %v", err)
		return
	}
	if err := os.Rename(tmp, p); err != nil {
		log.Printf("miner: mined balance replace failed: %v", err)
		if removeErr := os.Remove(tmp); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("miner: mined balance temp cleanup failed: %v", removeErr)
		}
	}
}

// resolveMinerEntryDifficultyV1 keeps every in-process miner entrypoint on
// the same network-profile and effective-difficulty policy.
func resolveMinerEntryDifficultyV1(
	cfg *config.Config,
	bc *blockchain.Blockchain,
	requestedDifficulty int,
) (int, error) {
	return resolveRuntimeDifficultyV1(
		cfg,
		bc,
		requestedDifficulty,
	)
}

// Start documents the QuantumCoin component used by this module.
// Start: sürekli kazıma döngüsünü başlatır
// An already-active call is idempotent and does not apply new options. Call
// StartExclusiveV1 when initialization must install a genuinely new loop.
func Start(
	bc *blockchain.Blockchain,
	minerAddress string,
	difficulty int,
	opts ...Options,
) error {
	return startPreparedV1(bc, minerAddress, difficulty, false, func() (Options, error) {
		return MergeOptionsV1(opts...), nil
	})
}

func startPreparedV1(bc *blockchain.Blockchain, minerAddress string, difficulty int, requireIdle bool, prepare func() (Options, error)) error {
	minerLifecycleMu.Lock()
	defer minerLifecycleMu.Unlock()
	if bc == nil {
		return errors.New(
			"miner: blockchain is nil",
		)
	}

	minerAddress = strings.TrimSpace(
		minerAddress,
	)
	if minerAddress == "" {
		return errors.New(
			"miner: miner address required",
		)
	}

	if IsActive() {
		if requireIdle {
			return ErrMinerAlreadyActiveV1
		}
		return nil
	}

	effectiveDifficulty, err :=
		resolveMinerEntryDifficultyV1(
			config.Current(),
			bc,
			difficulty,
		)
	if err != nil {
		return fmt.Errorf(
			"miner start difficulty: %w",
			err,
		)
	}

	merged, err := prepare()
	if err != nil {
		return fmt.Errorf("miner preparation: %w", err)
	}

	state.bc = bc
	state.address = minerAddress
	state.requestedDifficulty = difficulty
	state.difficulty = effectiveDifficulty
	state.opts = merged
	state.stopCh = make(chan struct{})
	state.doneCh = make(chan struct{})
	state.stopping = false
	state.step = 0
	state.hashrate = 0

	if merged.Animate {
		state.effect = NewEffect("QC")
	} else {
		state.effect = nil
	}

	state.active.Store(true)
	go loop(state.stopCh, state.doneCh)

	return nil
}

// Stop documents the QuantumCoin component used by this module.
// Stop: sürekli kazımayı durdurur
func Stop() {
	minerLifecycleMu.Lock()
	if !state.active.Load() {
		minerLifecycleMu.Unlock()
		return
	}
	done := state.doneCh
	if !state.stopping {
		state.stopping = true
		close(state.stopCh)
	}
	minerLifecycleMu.Unlock()
	<-done
}

// IsActive documents the QuantumCoin component used by this module.
// IsActive: çalışıyor mu?
func IsActive() bool {
	return state.active.Load()
}

// MineOne documents the QuantumCoin component used by this module.
// MineOne: tek seferlik blok kazı (paylaşılan bc ile)
func MineOne(
	bc *blockchain.Blockchain,
	address string,
	difficulty int,
) (*blockchain.Block, error) {
	if bc == nil {
		return nil, errors.New(
			"miner: blockchain is nil",
		)
	}

	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New(
			"miner: miner address required",
		)
	}

	effectiveDifficulty, err :=
		resolveMinerEntryDifficultyV1(
			config.Current(),
			bc,
			difficulty,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"miner mine one difficulty: %w",
			err,
		)
	}

	effect := NewEffect("QC")

	for step := 0; step < 10; step++ {
		effect.Frame(
			step,
			bc.BestHeightSnapshot()+1,
			effectiveDifficulty,
			0,
		)

		time.Sleep(
			80 * time.Millisecond,
		)
	}

	effect.Clear()

	candidateHeight := bc.BestHeightSnapshot() + 1
	reward := bc.ExpectedSubsidyForHeightSnapshot(
		candidateHeight,
	)

	startedAt := time.Now()

	block, err := bc.MineBlock(
		address,
		effectiveDifficulty,
	)

	elapsed := time.Since(startedAt)

	if err != nil {
		return nil, err
	}

	LogBlock(block)

	fmt.Printf(
		"✨ Reward: %d QC (elapsed %.2fs)\n",
		reward,
		elapsed.Seconds(),
	)

	showSplitInfoPreview(
		reward,
	)
	TrackMiner(address)

	addMinedBalance(
		address,
		reward,
	)

	return block, nil
}

// ---- iç döngü ----
func loop(stop <-chan struct{}, done chan struct{}) {
	defer func() {
		minerLifecycleMu.Lock()
		defer minerLifecycleMu.Unlock()
		if state.effect != nil {
			state.effect.Clear()
		}
		state.active.Store(false)
		close(done)
	}()

	for {
		select {
		case <-stop:
			return
		default:
		}

		effectiveDifficulty, err :=
			resolveMinerEntryDifficultyV1(
				config.Current(),
				state.bc,
				state.requestedDifficulty,
			)
		if err != nil {
			if state.opts.OnError != nil {
				state.opts.OnError(
					fmt.Errorf(
						"miner loop difficulty: %w",
						err,
					),
				)
			}

			time.Sleep(
				450 * time.Millisecond,
			)

			continue
		}

		state.difficulty = effectiveDifficulty

		if state.opts.Animate &&
			state.effect != nil {
			nextHeight :=
				state.bc.BestHeightSnapshot() + 1

			state.effect.Frame(
				state.step,
				nextHeight,
				effectiveDifficulty,
				state.hashrate,
			)

			state.step++
		}

		candidateHeight := state.bc.BestHeightSnapshot() + 1
		reward := state.bc.ExpectedSubsidyForHeightSnapshot(
			candidateHeight,
		)

		startedAt := time.Now()

		block, err := state.bc.MineBlock(
			state.address,
			effectiveDifficulty,
		)

		duration := time.Since(startedAt)

		state.hashrate = estimateHashrate(
			duration,
			effectiveDifficulty,
		)

		if err != nil {
			if state.opts.OnError != nil {
				state.opts.OnError(err)
			}

			time.Sleep(
				450 * time.Millisecond,
			)

			continue
		}

		status := MiningStatus{
			BlockHeight: block.Index,
			BlockHash:   block.Hash,
			Reward:      reward,
			Timestamp:   time.Now(),
		}

		if state.effect != nil {
			state.effect.Clear()
		}

		fmt.Printf(
			"🚀 New block #%d mined by %s  (hash=%x, t=%.2fs)\n",
			block.Index,
			state.address,
			block.Hash,
			duration.Seconds(),
		)

		if state.opts.Interval > 0 {
			if state.opts.Animate {
				consolefx.SpinFor(state.opts.Interval)
			} else {
				time.Sleep(state.opts.Interval)
			}
		}

		fmt.Printf(
			"💰 Reward: %d QC\n",
			status.Reward,
		)

		showSplitInfoPreview(
			status.Reward,
		)
		TrackMiner(state.address)

		addMinedBalance(
			state.address,
			status.Reward,
		)

		if state.opts.OnBlock != nil {
			state.opts.OnBlock(
				block,
				status,
			)
		}

		if state.opts.Broadcaster != nil {
			state.opts.Broadcaster(block)
		}

		if state.opts.Interval > 0 {
			time.Sleep(
				state.opts.Interval,
			)
		}

		if state.opts.OnTick != nil {
			state.opts.OnTick()
		}
	}
}

// LogBlock documents the QuantumCoin component used by this module.
// LogBlock: basit log çıktısı
func LogBlock(b *blockchain.Block) {
	log.Printf("🚀 New block: idx=%d hash=%x", b.Index, b.Hash)
}

// ---- Retired annual mining bonus compatibility ----

// checkYearlyBonus is retained only for source compatibility.
// QMA-0047: the annual CMD mining QC bonus is cancelled.
// This hook performs no payment, mint, logging or NFT award.
// Telegram monthly NFT processing is a separate program.
func checkYearlyBonus(address string) {
	_ = address
}

// Report the actual recipient contract; no unsupported stake/burn split is implied.
func showSplitInfoPreview(baseReward int) {
	if baseReward <= 0 {
		return
	}
	fmt.Println("[mining] coinbase pays the miner the consensus subsidy plus validated fees; staking/dev/burn/community block splits are inactive")
}

// ---- Geriye dönük uyumluluk katmanı ----

func SetGlobalBlockchain(bc *blockchain.Blockchain) { globalBC.Store(bc) }

func StartMining(
	minerAddress string,
	animationUpdate func(status MiningStatus),
) {
	chain := globalBC.Load()
	if IsActive() || chain == nil {
		return
	}

	err := Start(
		chain,
		minerAddress,
		0,
		Options{
			OnBlock: func(
				_ *blockchain.Block,
				status MiningStatus,
			) {
				if animationUpdate != nil {
					animationUpdate(status)
				}
			},
			Interval: 1500 * time.Millisecond,
			Animate:  true,
		},
	)
	if err != nil {
		log.Printf(
			"miner StartMining failed: %v",
			err,
		)
	}
}
func StopMining()          { Stop() }
func IsMiningActive() bool { return IsActive() }
