// miner/miner.go
package miner

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"quantumcoin/animation"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	internalpkg "quantumcoin/internal" // alias
)

// MiningStatus: dışa raporlanan durum
type MiningStatus struct {
	BlockHeight int
	BlockHash   []byte
	Reward      int
	Timestamp   time.Time
}

// Options: başlatma seçenekleri ve geri çağrılar
type Options struct {
	OnBlock     func(b *blockchain.Block, status MiningStatus)
	OnError     func(err error)
	OnTick      func()
	Interval    time.Duration
	Broadcaster func(b *blockchain.Block)
	Animate     bool
}

// ---- iç durum ----
var (
	state         minerState
	globalBC      *blockchain.Blockchain // geriye dönük uyumluluk
	yearBonusLock sync.Mutex
	yearlyGiven   = map[string]int{} // adres -> en son bonus verilen yıl idx
)

type minerState struct {
	active     atomic.Bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	bc         *blockchain.Blockchain
	address    string
	difficulty int
	opts       Options
}

// Start: sürekli kazıma döngüsünü başlatır
func Start(bc *blockchain.Blockchain, minerAddress string, difficulty int, opts ...Options) error {
	if bc == nil {
		return errors.New("miner: blockchain is nil")
	}
	if minerAddress == "" {
		return errors.New("miner: miner address required")
	}
	if IsActive() {
		return nil
	}

	merged := Options{Interval: 1200 * time.Millisecond, Animate: true}
	if len(opts) > 0 {
		o := opts[0]
		if o.OnBlock != nil {
			merged.OnBlock = o.OnBlock
		}
		if o.OnError != nil {
			merged.OnError = o.OnError
		}
		if o.OnTick != nil {
			merged.OnTick = o.OnTick
		}
		if o.Interval > 0 {
			merged.Interval = o.Interval
		}
		if o.Broadcaster != nil {
			merged.Broadcaster = o.Broadcaster
		}
		merged.Animate = o.Animate
	}

	state.bc = bc
	state.address = minerAddress
	state.difficulty = difficulty
	state.opts = merged
	state.stopCh = make(chan struct{})

	state.active.Store(true)
	state.wg.Add(1)
	go loop()
	return nil
}

// Stop: sürekli kazımayı durdurur
func Stop() {
	if !IsActive() {
		return
	}
	close(state.stopCh)
	state.wg.Wait()
	state.active.Store(false)
}

func IsActive() bool { return state.active.Load() }

// MineOne: tek seferlik blok kazı
func MineOne(bc *blockchain.Blockchain, address string, difficulty int) (*blockchain.Block, error) {
	if bc == nil {
		return nil, errors.New("miner: blockchain is nil")
	}
	if address == "" {
		return nil, errors.New("miner: miner address required")
	}

	animation.ShowMiningAnimation()

	block, err := bc.MineBlock(address, difficulty)
	if err != nil {
		return nil, err
	}
	LogBlock(block)
	animation.ShowRewardEffect(float64(blockchain.GetCurrentReward()))
	showSplitInfoPreview()
	checkYearlyBonus(address)
	return block, nil
}

// ---- iç döngü ----
func loop() {
	defer state.wg.Done()
	step := 0

	for {
		select {
		case <-state.stopCh:
			return
		default:
		}

		if state.opts.Animate {
			h := state.bc.GetBestHeight()
			animation.ShowMiningFrame(step, 10, 0, h+1, state.difficulty)
			step++
		}

		block, err := state.bc.MineBlock(state.address, state.difficulty)
		if err != nil {
			if state.opts.OnError != nil {
				state.opts.OnError(err)
			}
			time.Sleep(450 * time.Millisecond)
		} else {
			status := MiningStatus{
				BlockHeight: block.Index,
				BlockHash:   block.Hash,
				Reward:      blockchain.GetCurrentReward(),
				Timestamp:   time.Now(),
			}
			animation.ShowRewardEffect(float64(status.Reward))
			showSplitInfoPreview()
			checkYearlyBonus(state.address)

			if state.opts.OnBlock != nil {
				state.opts.OnBlock(block, status)
			}
			if state.opts.Broadcaster != nil {
				state.opts.Broadcaster(block)
			}
			if state.opts.Interval > 0 {
				time.Sleep(state.opts.Interval)
			}
		}

		if state.opts.OnTick != nil {
			state.opts.OnTick()
		}
	}
}

// LogBlock: basit log
func LogBlock(b *blockchain.Block) {
	log.Printf("🚀 New block: idx=%d hash=%x", b.Index, b.Hash)
}

// ---- Yıllık bonus + NFT tetikleyici ----
func checkYearlyBonus(address string) {
	cfg := config.Current()
	if cfg == nil || cfg.GenesisUnix <= 0 {
		return
	}
	now := time.Now().Unix()
	yearIdx := int((now - cfg.GenesisUnix) / (365 * 24 * 60 * 60))
	if yearIdx < 0 {
		yearIdx = 0
	}

	yearBonusLock.Lock()
	defer yearBonusLock.Unlock()
	if last, ok := yearlyGiven[address]; ok && last >= yearIdx {
		return // bu yıl zaten verilmiş
	}

	// kayıt amaçlı bonus (bakiyeyi değiştirmez)
	internalpkg.GiveBonus(address, "Yearly", 100, "Annual miner bonus", "")

	// NFT + görsel
	GrantNFTReward(address)
	animation.ShowSparkle("Annual 100 QC + Rare NFT")

	yearlyGiven[address] = yearIdx
}

// Ödül bölüşümü önizlemesi (bilgi amaçlı)
func showSplitInfoPreview() {
	cfg := config.Current()
	if cfg == nil || cfg.InitialReward <= 0 {
		return
	}
	base := float64(blockchain.GetCurrentReward())
	if base <= 0 {
		return
	}
	miner := base * float64(cfg.RewardPctMiner) / 100.0
	stake := base * float64(cfg.RewardPctStake) / 100.0
	dev := base * float64(cfg.RewardPctDev) / 100.0
	burn := base * float64(cfg.RewardPctBurn) / 100.0
	remain := base - (miner + stake + dev + burn)
	if remain < 0 {
		remain = 0
	}
	animation.ShowSplitInfo(miner, stake, dev, burn, remain)
}

// ---- Geriye dönük uyumluluk ----
func SetGlobalBlockchain(bc *blockchain.Blockchain) { globalBC = bc }

func StartMining(minerAddress string, animationUpdate func(status MiningStatus)) {
	if IsActive() || globalBC == nil {
		return
	}
	_ = Start(globalBC, minerAddress, 16, Options{
		OnBlock: func(b *blockchain.Block, st MiningStatus) {
			if animationUpdate != nil {
				animationUpdate(st)
			}
		},
		Interval: 1500 * time.Millisecond,
		Animate:  true,
	})
}

func StopMining()          { Stop() }
func IsMiningActive() bool { return IsActive() }
