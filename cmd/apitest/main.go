// Command apitest previously launched a second, package-api HTTP server.
//
// standalone HTTP retirement retired that architecture. Run the canonical QuantumCoin node
// when an HTTP API is required.
package main

import "log"

func main() {
	log.Fatal(
		"cmd/apitest: standalone package-api HTTP server retired; run the canonical QuantumCoin node API",
	)
}
