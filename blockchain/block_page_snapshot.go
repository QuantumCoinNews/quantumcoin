package blockchain

// BlocksPageFromTipSnapshot returns a detached page whose offset is measured
// backwards from the current chain tip.
//
// offset=0, limit=20 returns the newest twenty blocks while preserving the
// canonical ascending block order inside the returned page.
//
// Total and page boundaries are computed while holding the same read lock so
// API pagination metadata and page contents describe one coherent chain view.
func (bc *Blockchain) BlocksPageFromTipSnapshot(
	offset int,
	limit int,
) ([]*Block, int) {
	if bc == nil {
		return []*Block{}, 0
	}

	if limit <= 0 {
		return []*Block{}, 0
	}

	if offset < 0 {
		offset = 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	total := len(bc.Blocks)

	if offset > total {
		offset = total
	}

	end := total - offset
	start := end - limit

	if start < 0 {
		start = 0
	}

	result := make(
		[]*Block,
		end-start,
	)

	for index := start; index < end; index++ {
		result[index-start] =
			cloneBlockV1(
				bc.Blocks[index],
			)
	}

	return result, total
}
