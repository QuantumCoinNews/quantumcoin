# Signed CLI transfers

`quantumcoin send <from> <to> <amount-QC>` loads the requested sender's existing
private key from the configured wallet store, builds the canonical transaction,
signs every input, verifies the signature and submits it through
`Blockchain.AddTransaction`. Broadcasting is attempted only after admission.

The store is selected by `wallet_file` in node configuration or `QC_WALLET_FILE`.
Relative node storage paths are anchored to the selected data directory; see
[CLI paths](CLI_PATHS_AND_OFFLINE_COMMANDS.md). The default is `wallet_data.json`.
Use a wallet already saved in that store. Offline `newaddr` saves a generated
wallet including its private key; use `--wallet-file` or `QC_WALLET_FILE` to
match a custom node store because offline creation does not read node config.
Sending never generates a wallet, substitutes the default wallet or rewrites
the wallet store. Private keys are not passed through command-line arguments
and are not printed in the transfer result.

The stored key must derive exactly the requested `from` address. A missing or
unreadable store, missing sender, malformed key or mismatched key rejects the
transfer before mempool admission and before the success/broadcast path.
Invalid addresses, invalid/unrepresentable amounts, insufficient or immature
funds, invalid signatures and conflicting spends also reject the transfer.

The amount is a decimal QC string, using the same
`NewTransactionFromQCAmountV1` conversion as the local wallet signer. There is no
floating-point or integer-only CLI parsing. The active consensus protocol
determines valid precision; an amount not exactly representable at that height
is rejected. Historical whole-QC transfers remain supported.

The HTTP API accepts only an already signed `{"transaction": ...}` envelope.
It rejects legacy private-key fields. Both entry points use the same canonical
mempool admission and mandatory signature and UTXO ownership validation.

For signing without admission, broadcasting or chain saving, run
`quantumcoin sign-send FROM TO AMOUNT` or `quantumcoin sign-burn FROM AMOUNT`.
These commands use the existing local wallet store and mandatory trusted-chain
startup validation. An optional `--key-stdin` reads a private key from bounded
standard input; never put a private key in arguments, environment variables or
HTTP bodies. Success emits `SIGNED_TRANSACTION_V1=` followed by the JSON
envelope. Bootstrap diagnostics can precede this marker. Consumers must require
a valid marker/envelope rather than treating process exit alone as success.
Submit the envelope unchanged to the authenticated send or burn endpoint.
See [API and wallet contracts](API_WALLET_AND_SERVICE_CONTRACTS.md).

Tests in `cli_send_signing_test.go` cover stored-owner signatures, absent and
corrupt stores, mismatched keys, exact atom amounts, invalid amounts, unsigned
admission rejection and an attacker signing a victim's outpoint. A child-process
test runs the actual CLI dispatcher and mandatory startup loader against a
temporary valid chain; it checks successful signing and missing-key failure,
including absence of private keys in command output. Test keys are temporary.

Successful admission is a local mempool result. It does not promise peer
delivery, confirmation or durable mempool persistence across process exit.
