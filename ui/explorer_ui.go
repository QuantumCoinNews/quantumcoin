package ui

import (
	"fmt"

	"quantumcoin/blockchain"
	"quantumcoin/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowExplorerWindow(a fyne.App, w fyne.Window, bc *blockchain.Blockchain) {
	w.SetTitle(i18n.T(CurrentLang, "explorer_title"))
	content := container.NewVBox()
	scroll := container.NewVScroll(content)

	if bc != nil {
		for _, block := range bc.Blocks {
			content.Add(widget.NewLabel(fmt.Sprintf(i18n.T(CurrentLang, "explorer_block"), block.Index, block.Miner, block.Hash, block.PrevHash)))
			for _, tx := range block.Transactions {
				content.Add(widget.NewLabel(fmt.Sprintf(i18n.T(CurrentLang, "explorer_tx"), tx.ID)))
				for _, out := range tx.Outputs {
					content.Add(widget.NewLabel(fmt.Sprintf(i18n.T(CurrentLang, "explorer_tx_out"), out.Amount)))
				}
			}
			content.Add(widget.NewSeparator())
		}
	} else {
		content.Add(widget.NewLabel("—"))
	}

	w.SetContent(scroll)
	w.Resize(fyne.NewSize(740, 520))
	w.Show()
}
