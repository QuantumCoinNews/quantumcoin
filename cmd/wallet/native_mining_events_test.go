package main

import (
	"strings"
	"testing"
)

func TestNativeMiningAcceptanceRefreshesWalletOnUIDispatchOnce(t *testing.T) {
	guiMining.Lock()
	oldLog := guiMining.log
	var message string
	guiMining.log = func(value string) { message = value }
	guiMining.Unlock()
	oldRefresh := walletRefreshHook
	t.Cleanup(func() { guiMining.Lock(); guiMining.log = oldLog; guiMining.Unlock(); walletRefreshHook = oldRefresh })
	inUI, refreshes, dispatches := false, 0, 0
	walletRefreshHook = func() {
		if !inUI {
			t.Fatal("wallet refresh escaped UI dispatcher")
		}
		refreshes++
	}
	accepted := nativeMiningAcceptanceHandler(func(action func()) { dispatches++; inUI = true; action(); inUI = false })
	accepted(12, "test-only-block-hash")
	if refreshes != 1 || dispatches != 1 || !strings.Contains(message, "Block 12 accepted: test-only-block-hash") {
		t.Fatal("accepted block did not update wallet once", refreshes, dispatches, message)
	}
	walletRefreshHook = nil
	accepted(13, "test-only-next-hash")
	if refreshes != 1 || dispatches != 2 {
		t.Fatal("absent wallet callback changed acceptance handling")
	}
}
