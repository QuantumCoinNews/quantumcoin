// Package walletcli handles local encrypted QC wallets. Secrets enter only via
// an explicit stdin JSON pipe; they are never arguments, logs or HTTP fields.
package walletcli

import (
	"encoding/json"
	"errors"
	"io"
	"quantumcoin/wallet"
)

type input struct {
	Password string `json:"password"`
	Words    string `json:"words"`
	Address  string `json:"address"`
}

func Run(action, path string, in io.Reader, out io.Writer) error {
	if path == "" {
		return errors.New("encrypted output path required; use the GUI for interactive recovery words")
	}
	raw, err := io.ReadAll(io.LimitReader(in, 8193))
	if err != nil || len(raw) > 8192 {
		return errors.New("invalid local wallet input")
	}
	defer clear(raw)
	var request input
	if err = json.Unmarshal(raw, &request); err != nil {
		return errors.New("provide local stdin JSON with password, and words/address for restore; never use command-line secrets")
	}
	var material *wallet.RecoveryMaterial
	switch action {
	case "create":
		if request.Words != "" || request.Address != "" {
			return errors.New("create does not accept old recovery words")
		}
		material, err = wallet.NewRecoveryMaterial()
	case "restore":
		material, err = wallet.RestoreRecoveryMaterial(request.Words, request.Address)
	default:
		return errors.New("unsupported wallet operation")
	}
	if err != nil {
		return err
	}
	blob, err := wallet.SealRecoveryMaterial(*material, request.Password)
	if err != nil {
		return err
	}
	if err = wallet.SaveRecoveryVaultNew(path, blob); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(map[string]string{"address": material.Address, "vault": path, "profile": material.Profile, "note": "Encrypted QC backup saved. Import into the QC GUI to reveal and record your recovery words. Existing files were not replaced."})
}
