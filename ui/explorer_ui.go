package ui

import (
	"fmt"
	"quantumcoin/blockchain"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowExplorerWindow(a fyne.App, w fyne.Window, bc *blockchain.Blockchain) {
	w.SetTitle("Blockchain Explorer")
	content := container.NewVBox()
	scroll := container.NewVScroll(content)
	if bc != nil {
		for _, block := range bc.Blocks {
			blockLabel := widget.NewLabel(fmt.Sprintf("Block #%d - Miner: %s\nHash: %x", block.Index, block.Miner, block.Hash))
			content.Add(blockLabel)
			for _, tx := range block.Transactions {
				txLabel := widget.NewLabel(fmt.Sprintf("  TxID: %x", tx.ID))
				content.Add(txLabel)
				for _, out := range tx.Outputs {
					outLabel := widget.NewLabel(fmt.Sprintf("    Amount: %d QC", out.Amount))
					content.Add(outLabel)
				}
			}
			content.Add(widget.NewSeparator())
		}
	} else {
		content.Add(widget.NewLabel("Blockchain verisi mevcut değil."))
	}
	w.SetContent(scroll)
	w.Resize(fyne.NewSize(700, 500))
	w.Show()
}
