// Retired fixed-seed demo. This command intentionally cannot create a wallet.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Fixed-seed demo retired. Use QC GUI create/restore, or qc_wallet_create / qc_wallet_restore with an encrypted output vault. Existing wallets are preserved.")
	os.Exit(1)
}
