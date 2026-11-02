package token

// Symbol returns a locked scalar state copy.
func (t *Token) Symbol() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.symbol
}

// Name returns a locked scalar state copy.
func (t *Token) Name() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.name
}

// Decimals returns a locked scalar state copy.
func (t *Token) Decimals() uint8 {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.decimals
}

// MaxSupply returns a locked scalar state copy.
func (t *Token) MaxSupply() uint64 {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.maxSupply
}

// TotalSupply returns a locked scalar state copy.
func (t *Token) TotalSupply() uint64 {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.totalSupply
}

// Owner returns a locked scalar state copy.
func (t *Token) Owner() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.owner
}

// Name returns a locked scalar state copy.
func (t *QC20Token) Name() string {
	if t == nil {
		return ""
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

// Symbol returns a locked scalar state copy.
func (t *QC20Token) Symbol() string {
	if t == nil {
		return ""
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.symbol
}

// Owner returns a locked scalar state copy.
func (t *QC20Token) Owner() string {
	if t == nil {
		return ""
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.owner
}

// MaxSupply returns a locked scalar state copy.
func (t *QC20Token) MaxSupply() uint64 {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.maxSupply
}

// TotalSupply returns a locked scalar state copy.
func (t *QC20Token) TotalSupply() uint64 {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalSupply
}

// Decimals returns a locked scalar state copy.
func (t *QC20Token) Decimals() uint8 {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.decimals
}

// Name returns a locked scalar state copy.
func (n *QC721Token) Name() string {
	if n == nil {
		return ""
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.name
}

// Symbol returns a locked scalar state copy.
func (n *QC721Token) Symbol() string {
	if n == nil {
		return ""
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.symbol
}

// Owner returns a locked scalar state copy.
func (n *QC721Token) Owner() string {
	if n == nil {
		return ""
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.owner
}

// BalancesSnapshotV1 returns a locked defensive state copy.
func (t *Token) BalancesSnapshotV1() map[string]uint64 {
	if t == nil {
		return make(map[string]uint64)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return cloneTokenUint64MapV1(t.balances)
}

// BalancesSnapshotV1 returns a locked defensive state copy.
func (t *QC20Token) BalancesSnapshotV1() map[string]uint64 {
	if t == nil {
		return make(map[string]uint64)
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return cloneTokenUint64MapV1(t.balances)
}

// AllowancesSnapshotV1 returns a locked defensive state copy.
func (t *QC20Token) AllowancesSnapshotV1() map[string]map[string]uint64 {
	if t == nil {
		return make(map[string]map[string]uint64)
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return cloneQC20AllowancesV1(t.allowances)
}

// OwnersSnapshotV1 returns a locked defensive state copy.
func (n *QC721Token) OwnersSnapshotV1() map[uint64]string {
	if n == nil {
		return make(map[uint64]string)
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return cloneQC721StringMapV1(n.owners)
}

// TokenURIsSnapshotV1 returns a locked defensive state copy.
func (n *QC721Token) TokenURIsSnapshotV1() map[uint64]string {
	if n == nil {
		return make(map[uint64]string)
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return cloneQC721StringMapV1(n.tokenURIs)
}
