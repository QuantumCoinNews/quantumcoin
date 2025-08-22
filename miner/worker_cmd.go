package miner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Work struct {
	Left   []byte
	Right  []byte
	Target *big.Int
	Height int
	Miner  string
}

type Backend interface {
	GetWork(ctx context.Context, address string) (*Work, error)
	Submit(ctx context.Context, w *Work, nonce uint64, hashHex string) (bool, error)
}

type WorkerConfig struct {
	Threads      int
	Address      string
	GreenOnFound bool
}

type Worker struct {
	backend   Backend
	cfg       WorkerConfig
	HashCount uint64
	found     uint64
}

func NewWorker(backend Backend, cfg WorkerConfig) *Worker {
	if cfg.Threads <= 0 {
		cfg.Threads = runtime.NumCPU()
	}
	return &Worker{backend: backend, cfg: cfg}
}
func (w *Worker) Found() uint64 { return atomic.LoadUint64(&w.found) }

func (w *Worker) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		work, err := w.backend.GetWork(ctx, w.cfg.Address)
		if err != nil || work == nil || work.Target == nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if err := w.solve(ctx, work); err != nil && ctx.Err() == nil {
			return err
		}
	}
}

func (w *Worker) solve(ctx context.Context, work *Work) error {
	var (
		wg     sync.WaitGroup
		stopCh = make(chan struct{})
		once   sync.Once
	)
	wg.Add(w.cfg.Threads)
	for i := 0; i < w.cfg.Threads; i++ {
		seed := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-stopCh:
					return
				default:
				}
				nonce := uint64(seed.Int63())
				h := hashCandidate(work.Left, nonce, work.Right)
				atomic.AddUint64(&w.HashCount, 1)
				if new(big.Int).SetBytes(h[:]).Cmp(work.Target) <= 0 {
					hashHex := hex.EncodeToString(h[:])
					once.Do(func() {
						close(stopCh)
						ok, _ := w.backend.Submit(ctx, work, nonce, hashHex)
						if ok {
							atomic.AddUint64(&w.found, 1)
							if w.cfg.GreenOnFound {
								fmt.Print("\x1b[32m")
							}
							fmt.Printf("\n[BLOK BULUNDU] h=%d nonce=%d hash=%s height=%d\n", w.HashCount, nonce, hashHex, work.Height)
							if w.cfg.GreenOnFound {
								fmt.Print("\x1b[0m")
							}
						} else {
							fmt.Printf("\n[GEÇERSİZ] nonce=%d hash=%s\n", nonce, hashHex)
						}
					})
					return
				}
			}
		}()
	}
	wg.Wait()
	return nil
}

// Zincirin prepareData kuralıyla uyumlu: SHA256(Left || strconv.Itoa(nonce) || Right)
func hashCandidate(left []byte, nonce uint64, right []byte) [32]byte {
	nb := []byte(strconv.Itoa(int(nonce)))
	buf := make([]byte, 0, len(left)+len(nb)+len(right))
	buf = append(buf, left...)
	buf = append(buf, nb...)
	buf = append(buf, right...)
	return sha256.Sum256(buf)
}
