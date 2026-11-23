//go:build js && wasm

// A local-only browser worker uses the exact native QC recovery implementation.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"syscall/js"
	"time"

	"quantumcoin/wallet"
)

type request struct {
	Action    string                        `json:"action"`
	Words     string                        `json:"words"`
	Address   string                        `json:"address"`
	Password  string                        `json:"password"`
	Vault     string                        `json:"vault"`
	PlayerID  string                        `json:"playerId"`
	Challenge *wallet.RecoveryLinkChallenge `json:"challenge"`
}

var active *wallet.RecoveryMaterial
var expiresAt int64

func process(raw string) any {
	if len(raw) > 32768 {
		return map[string]any{"ok": false, "error": "Wallet request is too large."}
	}
	var r request
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return map[string]any{"ok": false, "error": "Invalid local wallet request."}
	}
	result := map[string]any{"ok": true}
	switch r.Action {
	case "create", "restore":
		var material *wallet.RecoveryMaterial
		var err error
		if r.Action == "create" {
			material, err = wallet.NewRecoveryMaterial()
		} else {
			material, err = wallet.RestoreRecoveryMaterial(r.Words, r.Address)
		}
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		vault, err := wallet.SealRecoveryMaterial(*material, r.Password)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		active = nil // Saving and unlocking are separate explicit operations.
		result["address"] = material.Address
		result["vault"] = string(vault)
		if r.Action == "create" {
			result["words"] = material.Mnemonic
		}
	case "unlock", "reveal":
		material, err := wallet.OpenRecoveryMaterial([]byte(r.Vault), r.Password)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		result["address"] = material.Address
		if r.Action == "unlock" {
			active = material
			expiresAt = time.Now().Unix() + 300
		} else {
			result["words"] = material.Mnemonic
		}
	case "lock":
		active = nil
	case "link":
		if active == nil || time.Now().Unix() >= expiresAt || r.Challenge == nil {
			active = nil
			return map[string]any{"ok": false, "error": "Unlock your QC wallet first."}
		}
		proof, err := wallet.SignRecoveryLink(*active, *r.Challenge, r.PlayerID, time.Now().Unix())
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		result["proof"] = proof
	default:
		return map[string]any{"ok": false, "error": "Unsupported local wallet operation."}
	}
	return result
}

func main() {
	bridge := js.FuncOf(func(_ js.Value, args []js.Value) (output any) {
		defer func() {
			if recover() != nil {
				output = `{"ok":false,"error":"Local wallet operation failed."}`
			}
		}()
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return `{"ok":false,"error":"Invalid wallet request."}`
		}
		data, err := json.Marshal(process(args[0].String()))
		if err != nil {
			return `{"ok":false,"error":"Local wallet operation failed."}`
		}
		return string(data)
	})
	js.Global().Set("QCWalletCrypto", bridge)
	select {}
}
