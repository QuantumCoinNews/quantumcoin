package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func lockMiningContextV1(ctx context.Context, mu *sync.RWMutex) error {
	if ctx == nil {
		return fmt.Errorf("mining context is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if ctx.Done() == nil {
		mu.Lock()
		return nil
	}
	for {
		if mu.TryLock() {
			if err := ctx.Err(); err != nil {
				mu.Unlock()
				return err
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}
