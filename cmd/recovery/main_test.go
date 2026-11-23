package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"quantumcoin/wallet"
	"testing"
)

func invoke(t *testing.T, args []string, password string) (string, error) {
	t.Helper()
	in, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	if _, err := writer.Write([]byte(password)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = run(args, in, &out)
	return out.String(), err
}
func TestCLISealsAndRecoversWalletWithoutDisclosingSecrets(t *testing.T) {
	dir := t.TempDir()
	w := wallet.NewWallet()
	privateHex := w.ExportPrivateKeyHex()
	raw, _ := json.Marshal(map[string]string{"address": w.GetAddress(), "privateKeyHex": privateHex})
	source := filepath.Join(dir, "source.json")
	sealed := filepath.Join(dir, "sealed.qcwallet")
	recovered := filepath.Join(dir, "recovered.json")
	if err := os.WriteFile(source, raw, 0600); err != nil {
		t.Fatal(err)
	}
	password := "synthetic-cli-password-only"
	sealArgs := []string{"seal-wallet", "--source", source, "--target", sealed, "--password-stdin"}
	output, err := invoke(t, sealArgs, password+"\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(output), []byte(password)) || bytes.Contains([]byte(output), []byte(privateHex)) {
		t.Fatal("CLI exposed secret")
	}
	args := []string{"unseal-wallet", "--source", sealed, "--target", recovered, "--password-stdin"}
	if _, err := invoke(t, args, password); err == nil {
		t.Fatal("unseal lacked offline requirement")
	}
	if _, err := invoke(t, append(args, "--offline"), password); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(recovered)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("CLI recovery changed wallet", err)
	}
	if _, err := invoke(t, sealArgs, password); err == nil {
		t.Fatal("CLI overwrote encrypted backup")
	}
	if _, err := invoke(t, []string{"seal-wallet", "--source", source, "--target", filepath.Join(dir, "bad.qcwallet")}, password); err == nil {
		t.Fatal("explicit secret input not required")
	}
}
