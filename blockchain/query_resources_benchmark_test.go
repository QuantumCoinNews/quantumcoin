package blockchain

import (
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"testing"
)

func BenchmarkQueryTransactionLocationIndex(b *testing.B) {
	for _, size := range []int{128, 32768} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			chain := &Blockchain{}
			for i := range size {
				id := make([]byte, 32)
				binary.BigEndian.PutUint64(id, uint64(i+1))
				chain.Blocks = append(chain.Blocks, &Block{Index: i, Transactions: []*Transaction{{ID: id}}})
			}
			id := hex.EncodeToString(chain.Blocks[size-1].Transactions[0].ID)
			chain.TransactionLocationSnapshot(id) // Exclude the one-time index build.
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if !chain.TransactionLocationSnapshot(id).InBlock {
					b.Fatal("lost indexed transaction")
				}
			}
		})
	}
}

func BenchmarkQueryBlockSummaryTransactionCardinality(b *testing.B) {
	for _, size := range []int{1, 16384} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			chain := &Blockchain{}
			txs := make([]*Transaction, size)
			for i := range 200 {
				chain.Blocks = append(chain.Blocks, &Block{Index: i, Hash: make([]byte, 32), Transactions: txs})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				page, total := chain.BlockSummaryPageFromTipSnapshot(0, 20)
				if len(page) != 20 || total != 200 || page[0].TxCount != size {
					b.Fatal("bad summary page")
				}
			}
		})
	}
}

func BenchmarkQueryBlockHashIndex(b *testing.B) {
	for _, size := range []int{128, 32768} {
		for _, hit := range []bool{false, true} {
			b.Run(strconv.Itoa(size)+"/hit="+strconv.FormatBool(hit), func(b *testing.B) {
				chain := &Blockchain{}
				for i := range size {
					hash := make([]byte, 32)
					binary.BigEndian.PutUint64(hash, uint64(i+1))
					chain.Blocks = append(chain.Blocks, &Block{Index: i, Hash: hash})
				}
				hash := make([]byte, 32)
				if hit {
					copy(hash, chain.Blocks[size-1].Hash)
				}
				chain.BlockByHashSnapshot(hash)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if (chain.BlockByHashSnapshot(hash) != nil) != hit {
						b.Fatal("incorrect block lookup")
					}
				}
			})
		}
	}
}
