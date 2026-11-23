// Package wallet contains QuantumCoin project code for this module.
package wallet

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"quantumcoin/utils"
)

/*
ImportPrivateKeyHex şu 3 formatı kabul eder:
1) SEC1 EC PRIVATE KEY (x509.MarshalECPrivateKey → DER → hex)
2) PKCS#8 PrivateKey (x509.MarshalPKCS8PrivateKey → DER → hex)
3) Ham 32 bayt secp256k1 anahtarı (sadece D; DER değil)
*/
func ImportPrivateKeyHex(h string) (*ecdsa.PrivateKey, error) {
	raw, err := hex.DecodeString(h)
	if err != nil {
		return nil, err
	}

	// 1) SEC1
	if pk, err := x509.ParseECPrivateKey(raw); err == nil {
		return pk, nil
	}

	// 2) PKCS#8
	if any, err := x509.ParsePKCS8PrivateKey(raw); err == nil {
		if pk, ok := any.(*ecdsa.PrivateKey); ok {
			return pk, nil
		}
		return nil, errors.New("pkcs8 is not ecdsa")
	}

	// 3) Ham 32 bayt (secp256k1)
	if len(raw) == 32 {
		return utils.ImportRawSecpKeyV1(raw)
	}

	return nil, errors.New("unknown private key format")
}
