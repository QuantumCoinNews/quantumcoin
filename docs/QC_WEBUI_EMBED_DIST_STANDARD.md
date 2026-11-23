# Browser balance viewer and embedded node UI

Document ID: `QC_WEBUI_EMBED_DIST_STANDARD_V1`\
Evidence stage: **Tested**. See [scope and release status](POLICY_EVIDENCE_REGISTER.md).

The node serves the files listed in `webui/embed.go`. The landing is
informational and the browser miner is an operator tool. Both are served
directly by the Go node.

The independent React balance viewer lives in `webwallet/`. On Node 22 with npm
10, enter that directory and run:

```sh
npm ci
npm test
npm run build
```

Distribute the resulting `webwallet/dist/` on a dedicated trusted origin with
same-origin forwarding for `GET /api/network/label` and
`GET /api/wallet/balance/{address}`. The balance endpoint requires the node's API
token in the `X-QC-API-KEY` header. Use HTTPS for remote access. The development
server binds to 127.0.0.1 and proxies these two paths to 127.0.0.1:8082.

Use exact `balanceQc` and `spendableQc` strings from the response. The viewer
clears the token field on submit and keeps wallet creation and signing with the
native wallet or CLI. The node landing and independent viewer each show network
identity from the same public API.

Keep npm's lockfile and source files in Git. Build output and dependencies are
generated. Compare the SHA-256 values of the output tree when verifying two
clean builds on the same toolchain and platform.
