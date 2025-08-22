package miner

import (
	"context"
	"crypto/rand"
	"math/big"
)

// Basit bir sahte backend: rastgele iş (Left/Right) verir,
// hedefi (Target) kolay ayarlar ve Submit'te sanity check yapar.
type mockBackend struct {
	target *big.Int
	height int
}

func NewMockBackend(addr string) Backend {
	// Yaklaşık 22 bit zorluk (kolay olsun).
	bits := 22
	t := big.NewInt(1)
	t.Lsh(t, uint(256-bits))
	return &mockBackend{target: t, height: 0}
}

func (m *mockBackend) GetWork(_ context.Context, address string) (*Work, error) {
	left := make([]byte, 64)
	right := make([]byte, 48)
	_, _ = rand.Read(left)
	_, _ = rand.Read(right)

	m.height++

	return &Work{
		Left:   left,
		Right:  append(right, []byte(address)...), // küçük karışım
		Target: new(big.Int).Set(m.target),
		Height: m.height,
		Miner:  address,
	}, nil
}

func (m *mockBackend) Submit(_ context.Context, w *Work, nonce uint64, _ string) (bool, error) {
	// Worker zaten hedefe göre kontrol etti; yine de sanity check:
	h := hashCandidate(w.Left, nonce, w.Right)
	return new(big.Int).SetBytes(h[:]).Cmp(w.Target) <= 0, nil
}
