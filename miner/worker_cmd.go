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

	"quantumcoin/blockchain"
)

type Work struct {
	Left     []byte
	Right    []byte
	Target   *big.Int
	Height   int
	Miner    string
	Template *blockchain.Block
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

func (w *Worker) solve(
	ctx context.Context,
	work *Work,
) error {
	if work == nil {
		return fmt.Errorf("worker: nil work")
	}

	if work.Target == nil {
		return fmt.Errorf("worker: nil work target")
	}

	var (
		wg       sync.WaitGroup
		stopCh   = make(chan struct{})
		once     sync.Once
		resultCh = make(chan error, 1)
	)

	wg.Add(w.cfg.Threads)

	for i := 0; i < w.cfg.Threads; i++ {
		seed := rand.New(
			rand.NewSource(
				time.Now().UnixNano() + int64(i),
			),
		)

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

				meetsTarget, hash, err :=
					hashWorkCandidateV1(
						work,
						nonce,
					)

				atomic.AddUint64(
					&w.HashCount,
					1,
				)

				if err != nil {
					once.Do(func() {
						close(stopCh)

						resultCh <- fmt.Errorf(
							"worker candidate hash: %w",
							err,
						)
					})

					return
				}

				if !meetsTarget {
					continue
				}

				hashHex := hex.EncodeToString(
					hash[:],
				)

				once.Do(func() {
					close(stopCh)

					submitted, submitErr :=
						w.backend.Submit(
							ctx,
							work,
							nonce,
							hashHex,
						)

					if submitErr != nil {
						resultCh <- fmt.Errorf(
							"worker submit: %w",
							submitErr,
						)

						return
					}

					if submitted {
						atomic.AddUint64(
							&w.found,
							1,
						)

						if w.cfg.GreenOnFound {
							fmt.Print("\x1b[32m")
						}

						fmt.Printf(
							"\n[BLOK BULUNDU] h=%d nonce=%d hash=%s height=%d\n",
							atomic.LoadUint64(
								&w.HashCount,
							),
							nonce,
							hashHex,
							work.Height,
						)

						if w.cfg.GreenOnFound {
							fmt.Print("\x1b[0m")
						}
					} else {
						fmt.Printf(
							"\n[GEÇERSİZ] nonce=%d hash=%s\n",
							nonce,
							hashHex,
						)
					}
				})

				return
			}
		}()
	}

	wg.Wait()

	select {
	case err := <-resultCh:
		return err
	default:
		return nil
	}
}

// hashWorkCandidateV1 uses the canonical blockchain PoW path when a fixed
// block template is attached. Template-free mock work keeps the legacy
// isolated hashing path until the backend is migrated.
func hashWorkCandidateV1(
	work *Work,
	nonce uint64,
) (bool, [32]byte, error) {
	var zero [32]byte

	if work == nil {
		return false, zero, fmt.Errorf(
			"worker: nil work",
		)
	}

	if work.Target == nil {
		return false, zero, fmt.Errorf(
			"worker: nil work target",
		)
	}

	if work.Template != nil {
		return blockchain.PoWHashMeetsTargetUint64V1(
			work.Template,
			nonce,
		)
	}
	hash := hashCandidate(
		work.Left,
		nonce,
		work.Right,
	)

	meetsTarget :=
		new(big.Int).
			SetBytes(hash[:]).
			Cmp(work.Target) <= 0

	return meetsTarget, hash, nil
}

// hashCandidate remains only for template-free mock/legacy work.
func hashCandidate(
	left []byte,
	nonce uint64,
	right []byte,
) [32]byte {
	nonceBytes := []byte(
		strconv.Itoa(int(nonce)),
	)

	buffer := make(
		[]byte,
		0,
		len(left)+len(nonceBytes)+len(right),
	)

	buffer = append(buffer, left...)
	buffer = append(buffer, nonceBytes...)
	buffer = append(buffer, right...)

	return sha256.Sum256(buffer)
}
