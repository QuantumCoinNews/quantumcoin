package ui

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"quantumcoin/i18n"
	"quantumcoin/wallet"
)

// WalletCLI: Terminal cüzdan arayüzü
type WalletCLI struct {
	Lang string
}

// Run: Cüzdan menüsünü başlat
func (cli *WalletCLI) Run() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println(i18n.T(cli.Lang, "wallet_menu"))
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		switch input {
		case "1":
			cli.createWallet()
		case "2":
			cli.viewAddress()
		case "3":
			fmt.Println(i18n.T(cli.Lang, "exit"))
			return
		default:
			fmt.Println(i18n.T(cli.Lang, "invalid_option"))
		}
	}
}

func (cli *WalletCLI) createWallet() {
	w := wallet.NewWallet()
	address := w.GetAddress()
	fmt.Println(i18n.T(cli.Lang, "new_wallet_created"), address)
	// İsteğe bağlı: wallet.SaveWalletToFile(w)
}

func (cli *WalletCLI) viewAddress() {
	fmt.Print(i18n.T(cli.Lang, "enter_pubkey"))
	reader := bufio.NewReader(os.Stdin)
	pubKeyStr, _ := reader.ReadString('\n')
	pubKeyStr = strings.TrimSpace(pubKeyStr)

	pubKeyBytes, err := hex.DecodeString(pubKeyStr)
	if err != nil {
		fmt.Println(i18n.T(cli.Lang, "invalid_pubkey_format"))
		return
	}
	address := wallet.HashAndEncode(pubKeyBytes)
	fmt.Println(i18n.T(cli.Lang, "wallet_address_is"), address)
}
