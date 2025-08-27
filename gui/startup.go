package main

import (
	"log"
	"os"
	"quantumcoin/blockchain"
	"quantumcoin/ui"
	"quantumcoin/wallet"

	"fyne.io/fyne/v2/app"
)

const blockchainFile = "chain_data.dat"

func main() {
	myApp := app.NewWithID("quantumcoin.app")
	mainWindow := myApp.NewWindow("QuantumCoin")

	wlt := wallet.LoadWalletFromFile()

	var bc *blockchain.Blockchain
	if _, err := os.Stat(blockchainFile); err == nil {
		bc, err = blockchain.LoadBlockchainFromFile(blockchainFile)
		if err != nil {
			log.Println("Blockchain dosyası okunamadı, yeni başlatılıyor:", err)
			bc = blockchain.NewBlockchain(50, 25500000)
		}
	} else {
		bc = blockchain.NewBlockchain(50, 25500000)
	}

	// Ana GUI arayüzünü başlat (hem cüzdan hem zincir parametreli)
	ui.LaunchMainUI(myApp, mainWindow, wlt, bc)

	myApp.Run()

	// Program kapanınca zinciri kaydet
	if err := bc.SaveToFile(blockchainFile); err != nil {
		log.Printf("Blockchain kaydedilemedi: %v", err)
	}
}
