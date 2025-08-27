// apitest.go
package main

import (
	"log"
	"quantumcoin/api"
	"quantumcoin/blockchain"
	"quantumcoin/config"
)

func main() {
	// config yükle (gerekirse stub)
	cfg := &config.Config{HTTPPort: "8081"}

	// boş blockchain nesnesi (sadece test için)
	bc := &blockchain.Blockchain{}

	api.Init(bc, nil, cfg)

	if err := api.StartHTTP(""); err != nil {
		log.Fatal(err)
	}
}
