package miner

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
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
	// Header is the node-issued canonical preimage with a trailing nonce.
	Header      []byte
	PrevHash    string
	GenesisHash string
	ExpiresAt   time.Time
}
type Backend interface {
	GetWork(ctx context.Context, address string) (*Work, error)
	Submit(ctx context.Context, w *Work, nonce uint64, hashHex string) (bool, error)
}

type WorkerConfig struct {
	Threads      int
	Address      string
	GreenOnFound bool
	// Zero selects a two-second tip check for backends supporting WorkCurrent.
	RefreshInterval time.Duration
	OnAccepted      func(height int, hash string)
}

type Worker struct {
	backend   Backend
	cfg       WorkerConfig
	HashCount uint64
	found     uint64
	initial   *Work
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
		work := w.initial
		w.initial = nil
		var err error
		if work == nil {
			work, err = w.backend.GetWork(ctx, w.cfg.Address)
		}
		if errors.Is(err, errNodeMiningDenied) {
			return err
		}
		if err != nil || work == nil || work.Target == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(300 * time.Millisecond):
			}
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
	// Both the refresh path and winner serialize their final decision. Once
	// submission starts, a tip update must not cancel an in-flight receipt.
	var phaseMu sync.Mutex
	submitting := false
	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()
	watchDone := make(chan struct{})
	if source, ok := w.backend.(interface {
		WorkCurrent(context.Context, *Work) (bool, error)
	}); ok {
		interval := w.cfg.RefreshInterval
		if interval <= 0 {
			interval = 2 * time.Second
		}
		go func() {
			defer close(watchDone)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-watchCtx.Done():
					return
				case <-ticker.C:
					current, err := source.WorkCurrent(watchCtx, work)
					if err != nil && !errors.Is(err, errNodeMiningDenied) && (work.ExpiresAt.IsZero() || time.Now().Before(work.ExpiresAt)) {
						continue
					}
					if current && err == nil {
						continue
					}
					phaseMu.Lock()
					if !submitting {
						once.Do(func() {
							close(stopCh)
							if errors.Is(err, errNodeMiningDenied) {
								resultCh <- err
							}
						})
					}
					phaseMu.Unlock()
					return
				}
			}
		}()
	} else {
		close(watchDone)
	}
	defer func() { stopWatch(); <-watchDone }()

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
					phaseMu.Lock()
					once.Do(func() {
						close(stopCh)

						resultCh <- fmt.Errorf(
							"worker candidate hash: %w",
							err,
						)
					})
					phaseMu.Unlock()

					return
				}

				if !meetsTarget {
					continue
				}

				hashHex := hex.EncodeToString(
					hash[:],
				)

				phaseMu.Lock()
				won := false
				once.Do(func() { submitting = true; close(stopCh); won = true })
				phaseMu.Unlock()
				if !won {
					return
				}

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
					if w.cfg.OnAccepted != nil {
						w.cfg.OnAccepted(work.Height, hashHex)
					}

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

	if len(work.Header) != 0 {
		if len(work.Header) < 8 || nonce > blockchain.MaxConsensusNonceV1 {
			return false, zero, fmt.Errorf("worker: invalid canonical header or nonce")
		}
		header := append([]byte(nil), work.Header...)
		binary.BigEndian.PutUint64(header[len(header)-8:], nonce)
		hash := sha256.Sum256(header)
		return new(big.Int).SetBytes(hash[:]).Cmp(work.Target) < 0, hash, nil
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
