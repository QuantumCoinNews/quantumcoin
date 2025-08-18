package miner

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qint "quantumcoin/internal"
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
	OnBlock     func(b *blockchain.Block, status MiningStatus) // Blok bulunduğunda
	OnError     func(err error)                                // Hata olduğunda
	OnTick      func()                                         // Her tur sonunda
	Interval    time.Duration                                  // Blok bulduktan sonra bekleme
	Broadcaster func(b *blockchain.Block)                      // P2P yayıncı (opsiyonel)
	Animate     bool                                           // Terminal animasyon
}

// ---- iç durum ----
var (
	state         minerState
	globalBC      *blockchain.Blockchain // geriye dönük uyumluluk
	yearBonusLock sync.Mutex
	yearlyGiven   = map[string]int{} // adres → son bonus yılı
)

type minerState struct {
	active     atomic.Bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	bc         *blockchain.Blockchain
	address    string
	difficulty int
	opts       Options

	// görsel/istatistik
	effect   *Effect
	step     int
	hashrate float64
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

	// opsiyonları birleştir
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
	if merged.Animate {
		state.effect = NewEffect("QC")
	}

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
	if state.effect != nil {
		state.effect.Clear()
	}
}

// IsActive: çalışıyor mu?
func IsActive() bool {
	return state.active.Load()
}

// MineOne: tek seferlik blok kazı (paylaşılan bc ile)
func MineOne(bc *blockchain.Blockchain, address string, difficulty int) (*blockchain.Block, error) {
	if bc == nil {
		return nil, errors.New("miner: blockchain is nil")
	}
	if address == "" {
		return nil, errors.New("miner: miner address required")
	}

	// kısa animasyon
	eff := NewEffect("QC")
	for i := 0; i < 10; i++ {
		eff.Frame(i, bc.GetBestHeight()+1, difficulty, 0)
		time.Sleep(80 * time.Millisecond)
	}
	eff.Clear()

	start := time.Now()
	block, err := bc.MineBlock(address, difficulty)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}
	LogBlock(block)

	rw := blockchain.GetCurrentReward()
	fmt.Printf("✨ Reward: %d QC (elapsed %.2fs)\n", rw, elapsed.Seconds())
	showSplitInfoPreview()
	checkYearlyBonus(address)
	TrackMiner(address) // sende varsa

	return block, nil
}

// ---- iç döngü ----
func loop() {
	defer state.wg.Done()

	for {
		select {
		case <-state.stopCh:
			return
		default:
		}

		// canlı animasyon karesi
		if state.opts.Animate && state.effect != nil {
			nextH := state.bc.GetBestHeight() + 1
			state.effect.Frame(state.step, nextH, state.difficulty, state.hashrate)
			state.step++
		}

		start := time.Now()
		block, err := state.bc.MineBlock(state.address, state.difficulty)
		dur := time.Since(start)

		// kaba hashrate kestirimi (sadece görsel)
		state.hashrate = estimateHashrate(dur, state.difficulty)

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
			if state.effect != nil {
				state.effect.Clear()
			}
			fmt.Printf("🚀 New block #%d mined by %s  (hash=%x, t=%.2fs)\n",
				block.Index, state.address, block.Hash, dur.Seconds())
			fmt.Printf("💰 Reward: %d QC\n", status.Reward)
			showSplitInfoPreview()
			checkYearlyBonus(state.address)
			TrackMiner(state.address) // sende varsa

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

// LogBlock: basit log çıktısı
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

	// 100 QC bonus (demo; kalıcı muhasebe sende)
	qint.GiveBonus(address, "Yearly", 100, "Annual miner bonus", "")

	// NFT hediyesi (stub)
	GrantNFTReward(address) // sende varsa

	fmt.Println("✨ Annual 100 QC + Rare NFT awarded!")
	yearlyGiven[address] = yearIdx
}

// ödül bölüşümü bilgisi (görsel/log; coinbase split chain tarafında uygulanıyor varsayımı)
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
	fmt.Printf("🧮 split preview → miner:%.2f stake:%.2f dev:%.2f burn:%.2f community:%.2f\n",
		miner, stake, dev, burn, remain)
}

// ---- Geriye dönük uyumluluk katmanı ----

func SetGlobalBlockchain(bc *blockchain.Blockchain) { globalBC = bc }

func StartMining(minerAddress string, animationUpdate func(status MiningStatus)) {
	if IsActive() || globalBC == nil {
		return
	}
	_ = Start(globalBC, minerAddress, config.Current().DefaultDifficultyBits, Options{
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
